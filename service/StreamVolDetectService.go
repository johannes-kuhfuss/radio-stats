// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type StreamVolDetector interface {
	Listen()
	ListenContext(context.Context)
}

type FfmpegRunner func(context.Context, string, ...string) ([]byte, error)

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

func runFfmpegCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
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
	} else {
		for _, v := range s.Cfg.StreamVolDetect.Urls {
			logger.Info(fmt.Sprintf("Starting to detect stream volume on %v", v))
		}
		s.Cfg.SetRunListen(true)
	}

	ticker := time.NewTicker(intervalSeconds(s.Cfg.StreamVolDetect.IntervalSec))
	defer ticker.Stop()
	for s.Cfg.ShouldRunListen() {
		select {
		case <-ctx.Done():
			s.Cfg.SetRunListen(false)
			return
		default:
		}
		for _, streamUrl := range s.Cfg.StreamVolDetect.Urls {
			s.ListenRun(streamUrl)
		}
		select {
		case <-ctx.Done():
			s.Cfg.SetRunListen(false)
			return
		case <-ticker.C:
		}
	}
}

func (s DefaultStreamVolDetectService) ListenRun(streamUrl string) {
	s.increaseDetectCount()
	lines := s.runFfmpeg(streamUrl)
	if lines != nil {
		s.updateVolMetrics(lines, streamUrl)
	}
}

func (s DefaultStreamVolDetectService) increaseDetectCount() {
	s.Cfg.IncStreamVolDetectCount()
	s.Cfg.Metrics.StreamVolDetectCount.Inc()
}

func (s DefaultStreamVolDetectService) updateVolMetrics(lines []string, streamUrl string) {
	for _, line := range lines {
		if strings.Contains(line, "mean_volume") {
			re := regexp.MustCompile(`[-]\d*[\.]\d`)
			allNums := re.FindAllString(line, -1)
			for _, num := range allNums {
				f, err := strconv.ParseFloat(num, 64)
				if err == nil {
					s.Cfg.RunTime.StreamVolumes.Lock()
					s.Cfg.RunTime.StreamVolumes.Vols[streamUrl] = f
					s.Cfg.RunTime.StreamVolumes.Unlock()
					s.Cfg.Metrics.StreamVolume.WithLabelValues(streamUrl).Set(f)
				}
			}
		}
	}
}

func (s DefaultStreamVolDetectService) runFfmpeg(streamUrl string) (lines []string) {
	if s.FfmpegRunner == nil {
		s.FfmpegRunner = runFfmpegCommand
	}
	ctx := context.Background()
	timeout := time.Duration(s.Cfg.StreamVolDetect.Duration+5) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, err := s.FfmpegRunner(ctx, s.Cfg.StreamVolDetect.FfmpegExe, "-t", strconv.Itoa(s.Cfg.StreamVolDetect.Duration), "-i", streamUrl, "-af", "volumedetect", "-f", "null", "/dev/null")
	if err != nil {
		logger.Error(fmt.Sprintf("Could not execute ffmpeg on URL %v: ", streamUrl), err)
		return nil
	}
	return strings.Split(string(out), "\n")
}
