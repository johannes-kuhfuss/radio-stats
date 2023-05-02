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
	if s.Cfg.StreamVolDetect.Url == "" {
		logger.Warn("No volume detection URL given. Not starting stream volume detection")
		s.Cfg.RunTime.RunListen = false
	} else {
		logger.Info(fmt.Sprintf("Starting to detect stream volume on %v", s.Cfg.StreamVolDetect.Url))
		s.Cfg.RunTime.RunListen = true
	}

	for s.Cfg.RunTime.RunListen == true {
		ListenRun(s)
		time.Sleep(time.Duration(s.Cfg.StreamVolDetect.IntervalSec) * time.Second)
	}
}

func ListenRun(s DefaultStreamVolDetectService) {
	s.Cfg.RunTime.StreamVolDetectCount++
	s.Cfg.Metrics.StreamVolDetectCount.Inc()
	lines := runFfmpeg(s)
	if lines != nil {
		updateMetrics(lines, s)
	}
}

func updateMetrics(lines []string, s DefaultStreamVolDetectService) {
	for _, line := range lines {
		if strings.Contains(line, "mean_volume") {
			re := regexp.MustCompile(`[-]\d*[\.]\d`)
			allNums := re.FindAllString(line, -1)
			for _, num := range allNums {
				f, err := strconv.ParseFloat(num, 64)
				if err == nil {
					s.Cfg.RunTime.StreamVolume = f
					s.Cfg.Metrics.StreamVolume.WithLabelValues(s.Cfg.StreamVolDetect.Url).Set(f)
				}
			}
		}
	}
}

func runFfmpeg(s DefaultStreamVolDetectService) []string {
	ctx := context.Background()
	timeout := time.Duration(s.Cfg.StreamVolDetect.Duration+5) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	cmd := exec.CommandContext(ctx, s.Cfg.StreamVolDetect.FfmpegExe, "-t", strconv.Itoa(s.Cfg.StreamVolDetect.Duration), "-i", s.Cfg.StreamVolDetect.Url, "-af", "volumedetect", "-f", "null", "/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		cancel()
		logger.Error("Could not execute ffmpeg: ", err)
		return nil
	}
	cancel()
	lines := strings.Split(string(out), "\n")
	return lines
}
