// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

const (
	ffmpegSampleRate = 48000
	rmsMetadataKey   = "lavfi.astats.Overall.RMS_level"
	peakMetadataKey  = "lavfi.astats.Overall.Peak_level"
	lufsMetadataKey  = "lavfi.r128.S"
)

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
	monitor := newStreamAudioMonitor(s, streamURL)
	monitor.reset()
	s.Cfg.Metrics.StreamVolRestarts.WithLabelValues(streamURL).Add(0)
	backoff := time.Second
	attempt := 0
	for ctx.Err() == nil {
		if attempt > 0 {
			s.Cfg.Metrics.StreamVolRestarts.WithLabelValues(streamURL).Inc()
		}
		attempt++
		s.Cfg.Metrics.StreamVolDetectorUp.WithLabelValues(streamURL).Set(0)
		err := s.FfmpegRunner(ctx, s.Cfg.StreamVolDetect.FfmpegExe, ffmpegArgs(streamURL), func(line string) {
			if monitor.handleLine(line) {
				backoff = time.Second
			}
		})
		s.Cfg.Metrics.StreamVolDetectorUp.WithLabelValues(streamURL).Set(0)
		if ctx.Err() != nil {
			return
		}

		monitor.reset()
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
		"-filter_complex", fmt.Sprintf(
			"[0:a]asplit=2[stats][loudness];[stats]aresample=%d,asetnsamples=n=%d:p=0,astats=metadata=1:reset=1:measure_perchannel=none:measure_overall=RMS_level+Peak_level,ametadata=mode=print[stats_out];[loudness]ebur128=metadata=1,ametadata=mode=print,anullsink",
			ffmpegSampleRate,
			ffmpegSampleRate,
		),
		"-map", "[stats_out]",
		"-f", "null", "-",
	}
}

func parseRMSLevel(line string) (float64, bool) {
	return parseMetadataValue(line, rmsMetadataKey)
}

func parseMetadataValue(line, key string) (float64, bool) {
	prefix := key + "="
	index := strings.Index(line, prefix)
	if index < 0 {
		return 0, false
	}
	valueText := strings.TrimSpace(line[index+len(prefix):])
	if fields := strings.Fields(valueText); len(fields) > 0 {
		valueText = fields[0]
	}
	if strings.EqualFold(valueText, "-inf") {
		return math.Inf(-1), true
	}
	level, err := strconv.ParseFloat(valueText, 64)
	return level, err == nil && !math.IsNaN(level)
}

type streamAudioMonitor struct {
	service *DefaultStreamVolDetectService
	url     string
	window  *volumeWindow
	silence *silenceTracker
}

func newStreamAudioMonitor(service DefaultStreamVolDetectService, streamURL string) *streamAudioMonitor {
	return &streamAudioMonitor{
		service: &service,
		url:     streamURL,
		window:  newVolumeWindow(service.Cfg.StreamVolDetect.Duration, service.Cfg.StreamVolDetect.IntervalSec),
		silence: newSilenceTracker(service.Cfg.StreamVolDetect.SilenceThresholdDB, service.Cfg.StreamVolDetect.SilenceDurationSec),
	}
}

// handleLine updates the metric represented by line and reports whether the
// line contained a valid audio measurement.
func (m *streamAudioMonitor) handleLine(line string) bool {
	if level, ok := parseMetadataValue(line, rmsMetadataKey); ok {
		m.markMeasurement()
		duration, silent := m.silence.observe(level)
		m.service.Cfg.Metrics.StreamSilenceDuration.WithLabelValues(m.url).Set(duration)
		m.service.Cfg.Metrics.StreamAudioSilent.WithLabelValues(m.url).Set(boolFloat(silent))
		if meanLevel, complete := m.window.add(level); complete {
			m.service.updateVolMetric(meanLevel, m.url)
		}
		return true
	}
	if level, ok := parseMetadataValue(line, peakMetadataKey); ok {
		m.markMeasurement()
		m.service.Cfg.Metrics.StreamAudioPeak.WithLabelValues(m.url).Set(level)
		return true
	}
	if loudness, ok := parseMetadataValue(line, lufsMetadataKey); ok {
		m.markMeasurement()
		m.service.Cfg.Metrics.StreamAudioLoudness.WithLabelValues(m.url).Set(loudness)
		return true
	}
	return false
}

func (m *streamAudioMonitor) markMeasurement() {
	m.service.Cfg.Metrics.StreamVolDetectorUp.WithLabelValues(m.url).Set(1)
	m.service.Cfg.Metrics.StreamVolLastSample.WithLabelValues(m.url).SetToCurrentTime()
}

func (m *streamAudioMonitor) reset() {
	m.window.reset()
	m.silence.reset()
	m.service.Cfg.Metrics.StreamVolDetectorUp.WithLabelValues(m.url).Set(0)
	m.service.Cfg.Metrics.StreamAudioSilent.WithLabelValues(m.url).Set(0)
	m.service.Cfg.Metrics.StreamSilenceDuration.WithLabelValues(m.url).Set(0)
}

func boolFloat(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

type silenceTracker struct {
	thresholdDB    float64
	holdSeconds    int
	currentSeconds int
}

func newSilenceTracker(thresholdDB float64, holdSeconds int) *silenceTracker {
	if holdSeconds < 1 {
		holdSeconds = 1
	}
	return &silenceTracker{thresholdDB: thresholdDB, holdSeconds: holdSeconds}
}

func (s *silenceTracker) observe(level float64) (float64, bool) {
	if level <= s.thresholdDB {
		s.currentSeconds++
	} else {
		s.currentSeconds = 0
	}
	return float64(s.currentSeconds), s.currentSeconds >= s.holdSeconds
}

func (s *silenceTracker) reset() {
	s.currentSeconds = 0
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
