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
	ffmpegSampleRate       = 48000
	maxFfmpegDiagnostics   = 4
	maxFfmpegDiagnosticLen = 512
	rmsMetadataKey         = "lavfi.astats.Overall.RMS_level"
	peakMetadataKey        = "lavfi.astats.Overall.Peak_level"
	lufsMetadataKey        = "lavfi.r128.S"
)

type StreamVolDetector interface {
	Listen()
	ListenContext(context.Context)
}

// FfmpegRunner runs ffmpeg until it exits or ctx is cancelled and passes each
// diagnostic line to onLine as soon as ffmpeg produces it.
type FfmpegRunner func(ctx context.Context, name string, args []string, onLine func(string)) error

type DefaultStreamVolDetectService struct {
	Cfg             *config.AppConfig
	FfmpegRunner    FfmpegRunner
	watchdogTimeout time.Duration
}

func NewStreamVolDetectService(cfg *config.AppConfig) DefaultStreamVolDetectService {
	watchdogTimeout := time.Duration(cfg.StreamVolDetect.FreshnessTimeoutSec) * time.Second
	if watchdogTimeout <= 0 {
		watchdogTimeout = 15 * time.Second
	}
	return DefaultStreamVolDetectService{
		Cfg:             cfg,
		FfmpegRunner:    runFfmpegCommand,
		watchdogTimeout: watchdogTimeout,
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
		err, stale, diagnostics := s.runFfmpegWithWatchdog(ctx, streamURL, monitor, func() {
			backoff = time.Second
		})
		s.Cfg.Metrics.StreamVolDetectorUp.WithLabelValues(streamURL).Set(0)
		if ctx.Err() != nil {
			return
		}

		monitor.invalidateMeasurements()
		monitor.reset()
		if stale {
			if diagnostics == "" {
				logger.Errorf("Restarting ffmpeg for URL %v after receiving no valid audio measurements for %v", streamURL, s.watchdogTimeout)
			} else {
				logger.Errorf("Restarting ffmpeg for URL %v after receiving no valid audio measurements for %v. ffmpeg: %v", streamURL, s.watchdogTimeout, diagnostics)
			}
		} else {
			if diagnostics == "" {
				logger.Errorf("ffmpeg exited for URL %v: %v", streamURL, err)
			} else {
				logger.Errorf("ffmpeg exited for URL %v: %v. ffmpeg: %v", streamURL, err, diagnostics)
			}
		}
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

// runFfmpegWithWatchdog cancels a single ffmpeg run when it remains alive but
// stops producing valid audio measurements. CommandContext then terminates the
// child process and listenStream starts a fresh one using its existing retry
// loop.
func (s DefaultStreamVolDetectService) runFfmpegWithWatchdog(
	ctx context.Context,
	streamURL string,
	monitor *streamAudioMonitor,
	onMeasurement func(),
) (error, bool, string) {
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()

	diagnostics := newFfmpegDiagnostics()
	measurements := make(chan struct{}, 1)
	watchdogDone := make(chan struct{})
	stale := make(chan struct{}, 1)
	go func() {
		defer close(watchdogDone)
		timer := time.NewTimer(s.watchdogTimeout)
		defer timer.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-measurements:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(s.watchdogTimeout)
			case <-timer.C:
				s.Cfg.Metrics.StreamVolDetectorUp.WithLabelValues(streamURL).Set(0)
				stale <- struct{}{}
				cancelRun()
				return
			}
		}
	}()

	err := s.FfmpegRunner(runCtx, s.Cfg.StreamVolDetect.FfmpegExe, ffmpegArgs(streamURL), func(line string) {
		if !monitor.handleLine(line) {
			diagnostics.observe(line)
			return
		}
		onMeasurement()
		select {
		case measurements <- struct{}{}:
		default:
		}
	})
	cancelRun()
	<-watchdogDone
	select {
	case <-stale:
		return err, true, diagnostics.String()
	default:
		return err, false, diagnostics.String()
	}
}

type ffmpegDiagnostics struct {
	lines    []string
	fallback string
}

func newFfmpegDiagnostics() *ffmpegDiagnostics {
	return &ffmpegDiagnostics{lines: make([]string, 0, maxFfmpegDiagnostics)}
}

func (d *ffmpegDiagnostics) observe(line string) {
	line = strings.TrimSpace(line)
	if line == "" || strings.Contains(line, "lavfi.") {
		return
	}
	if len(line) > maxFfmpegDiagnosticLen {
		line = line[:maxFfmpegDiagnosticLen]
	}
	d.fallback = line
	if !isFfmpegErrorLine(line) {
		return
	}
	if len(d.lines) == maxFfmpegDiagnostics {
		copy(d.lines, d.lines[1:])
		d.lines = d.lines[:maxFfmpegDiagnostics-1]
	}
	d.lines = append(d.lines, line)
}

func (d *ffmpegDiagnostics) String() string {
	if len(d.lines) > 0 {
		return strings.Join(d.lines, " | ")
	}
	return d.fallback
}

func isFfmpegErrorLine(line string) bool {
	lower := strings.ToLower(line)
	for _, marker := range []string{
		"error", "failed", "invalid", "not found", "timed out",
		"refused", "unreachable", "server returned", "i/o error",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
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

func (m *streamAudioMonitor) invalidateMeasurements() {
	unknown := math.NaN()
	m.service.Cfg.RunTime.StreamVolumes.Lock()
	m.service.Cfg.RunTime.StreamVolumes.Vols[m.url] = unknown
	m.service.Cfg.RunTime.StreamVolumes.Unlock()
	m.service.Cfg.Metrics.StreamVolume.WithLabelValues(m.url).Set(unknown)
	m.service.Cfg.Metrics.StreamAudioPeak.WithLabelValues(m.url).Set(unknown)
	m.service.Cfg.Metrics.StreamAudioLoudness.WithLabelValues(m.url).Set(unknown)
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
