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

type StreamVolDetectService interface {
	Listen()
}

type DefaultStreamVolDetectService struct {
	Cfg *config.AppConfig
}

func NewStreamVolDetectService(cfg *config.AppConfig) DefaultStreamVolDetectService {
	return DefaultStreamVolDetectService{
		Cfg: cfg,
	}
}

func (s DefaultStreamVolDetectService) Listen() {
	if len(s.Cfg.StreamVolDetect.Urls) == 0 {
		logger.Warn("No volume detection URLs given. Not starting stream volume detection")
		s.Cfg.RunTime.RunListen = false
	} else {
		for _, v := range s.Cfg.StreamVolDetect.Urls {
			logger.Info(fmt.Sprintf("Starting to detect stream volume on %v", v))
		}
		s.Cfg.RunTime.RunListen = true
	}

	for s.Cfg.RunTime.RunListen {
		for _, streamUrl := range s.Cfg.StreamVolDetect.Urls {
			go s.ListenRun(streamUrl)
		}
		time.Sleep(time.Duration(s.Cfg.StreamVolDetect.IntervalSec) * time.Second)
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
	s.Cfg.RunTime.StreamVolDetectCount++
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

func (s DefaultStreamVolDetectService) runFfmpeg(streamUrl string) []string {
	ctx := context.Background()
	timeout := time.Duration(s.Cfg.StreamVolDetect.Duration+5) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	cmd := exec.CommandContext(ctx, s.Cfg.StreamVolDetect.FfmpegExe, "-t", strconv.Itoa(s.Cfg.StreamVolDetect.Duration), "-i", streamUrl, "-af", "volumedetect", "-f", "null", "/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		cancel()
		logger.Error(fmt.Sprintf("Could not execute ffmpeg on URL %v: ", streamUrl), err)
		return nil
	}
	cancel()
	lines := strings.Split(string(out), "\n")
	return lines
}
