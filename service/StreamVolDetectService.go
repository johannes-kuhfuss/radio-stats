// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

const ffmpegSampleRate = 48000

var rmsLevelPattern = regexp.MustCompile(`lavfi\.astats\.Overall\.RMS_level=(-?(?:\d+(?:\.\d*)?|\.\d+|inf))`)

type StreamVolDetector interface {
	Listen()
	ListenContext(context.Context)
}

// FfmpegRunner runs ffmpeg until it exits or ctx is cancelled and passes each
// diagnostic line to onLine as soon as ffmpeg produces it.
type FfmpegRunner func(ctx context.Context, name string, args []string, onLine func(string)) error

type DefaultStreamVolDetectService struct {
	Cfg          *config.AppConfig
	FfmpegRunner FfmpegRunner
}

func NewStreamVolDetectService(cfg *config.AppConfig) DefaultStreamVolDetectService {
	return DefaultStreamVolDetectService{
		Cfg:          cfg,
		FfmpegRunner: runFfmpegCommand,
	}
}

func runFfmpegCommand(ctx context.Context, name string, args []string, onLine func(string)) error {
	cmd := exec.CommandContext(ctx, name, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	scanErr := scanLines(stderr, onLine)
	waitErr := cmd.Wait()
	if scanErr != nil {
		return scanErr
	}
	return waitErr
}

func scanLines(reader io.Reader, onLine func(string)) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		onLine(scanner.Text())
	}
	return scanner.Err()
}

func (s DefaultStreamVolDetectService) Listen() {
	s.ListenContext(context.Background())
}

func (s DefaultStreamVolDetectService) ListenContext(ctx context.Context) {
	if s.FfmpegRunner == nil {
		s.FfmpegRunner = runFfmpegCommand
	}
	if len(s.Cfg.StreamVolDetect.Urls) == 0 {
		logger.Warn("No volume detection URLs given. Not starting stream volume detection")
		s.Cfg.SetRunListen(false)
		return
	}

	workerCtx, cancelWorkers := context.WithCancel(ctx)
	var workers sync.WaitGroup
	for _, streamURL := range s.Cfg.StreamVolDetect.Urls {
		logger.Info(fmt.Sprintf("Starting to detect stream volume on %v", streamURL))
		workers.Add(1)
		go func() {
			defer workers.Done()
			s.listenStream(workerCtx, streamURL)
		}()
	}
	s.Cfg.SetRunListen(true)

	stopCheck := time.NewTicker(100 * time.Millisecond)
	defer stopCheck.Stop()
	for s.Cfg.ShouldRunListen() {
		select {
		case <-ctx.Done():
			s.Cfg.SetRunListen(false)
		case <-stopCheck.C:
		}
	}

	cancelWorkers()
	workers.Wait()
}

func (s DefaultStreamVolDetectService) listenStream(ctx context.Context, streamURL string) {
	window := newVolumeWindow(s.Cfg.StreamVolDetect.Duration, s.Cfg.StreamVolDetect.IntervalSec)
	backoff := time.Second
	for ctx.Err() == nil {
		err := s.FfmpegRunner(ctx, s.Cfg.StreamVolDetect.FfmpegExe, ffmpegArgs(streamURL), func(line string) {
			level, ok := parseRMSLevel(line)
			if !ok {
				return
			}
			if meanLevel, complete := window.add(level); complete {
				s.updateVolMetric(meanLevel, streamURL)
				backoff = time.Second
			}
		})
		if ctx.Err() != nil {
			return
		}

		window.reset()
		logger.Error(fmt.Sprintf("ffmpeg exited for URL %v: ", streamURL), err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 15*time.Second {
			backoff *= 2
		} else {
			backoff = 30 * time.Second
		}
	}
}

func ffmpegArgs(streamURL string) []string {
	return []string{
		"-nostdin",
		"-hide_banner",
		"-loglevel", "info",
		"-reconnect", "1",
		"-reconnect_streamed", "1",
		"-reconnect_at_eof", "1",
		"-reconnect_on_network_error", "1",
		"-i", streamURL,
		"-vn",
		"-af", fmt.Sprintf(
			"aresample=%d,asetnsamples=n=%d:p=0,astats=metadata=1:reset=1:measure_perchannel=none:measure_overall=RMS_level,ametadata=mode=print:key=lavfi.astats.Overall.RMS_level",
			ffmpegSampleRate,
			ffmpegSampleRate,
		),
		"-f", "null", "-",
	}
}

func parseRMSLevel(line string) (float64, bool) {
	match := rmsLevelPattern.FindStringSubmatch(line)
	if len(match) != 2 {
		return 0, false
	}
	if strings.EqualFold(match[1], "-inf") {
		return math.Inf(-1), true
	}
	level, err := strconv.ParseFloat(match[1], 64)
	return level, err == nil && !math.IsNaN(level)
}

func (s DefaultStreamVolDetectService) increaseDetectCount() {
	s.Cfg.IncStreamVolDetectCount()
	s.Cfg.Metrics.StreamVolDetectCount.Inc()
}

func (s DefaultStreamVolDetectService) updateVolMetric(level float64, streamURL string) {
	s.Cfg.RunTime.StreamVolumes.Lock()
	s.Cfg.RunTime.StreamVolumes.Vols[streamURL] = level
	s.Cfg.RunTime.StreamVolumes.Unlock()
	s.Cfg.Metrics.StreamVolume.WithLabelValues(streamURL).Set(level)
	s.increaseDetectCount()
}

// volumeWindow combines equally sized one-second RMS measurements. RMS values
// must be averaged as linear power rather than directly in decibels.
type volumeWindow struct {
	durationSeconds int
	skipSeconds     int
	samples         int
	skipRemaining   int
	powerSum        float64
}

func newVolumeWindow(durationSeconds, intervalSeconds int) *volumeWindow {
	if durationSeconds < 1 {
		durationSeconds = 1
	}
	if intervalSeconds < durationSeconds {
		intervalSeconds = durationSeconds
	}
	return &volumeWindow{
		durationSeconds: durationSeconds,
		skipSeconds:     intervalSeconds - durationSeconds,
	}
}

func (w *volumeWindow) add(level float64) (float64, bool) {
	if w.skipRemaining > 0 {
		w.skipRemaining--
		return 0, false
	}

	w.powerSum += math.Pow(10, level/10)
	w.samples++
	if w.samples < w.durationSeconds {
		return 0, false
	}

	meanLevel := 10 * math.Log10(w.powerSum/float64(w.samples))
	w.samples = 0
	w.powerSum = 0
	w.skipRemaining = w.skipSeconds
	return meanLevel, true
}

func (w *volumeWindow) reset() {
	w.samples = 0
	w.skipRemaining = 0
	w.powerSum = 0
}
