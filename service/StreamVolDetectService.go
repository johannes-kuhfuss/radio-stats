package service

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
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

var (
	mu             sync.Mutex
	deltaZeroCount int
)

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
			go ListenRun(s, streamUrl)
		}
		time.Sleep(time.Duration(s.Cfg.StreamVolDetect.IntervalSec) * time.Second)
	}
}

func ListenRun(s DefaultStreamVolDetectService, streamUrl string) {
	mu.Lock()
	s.Cfg.RunTime.StreamVolDetectCount++
	s.Cfg.Metrics.StreamVolDetectCount.Inc()
	mu.Unlock()
	lines := runFfmpeg(s, streamUrl)
	if lines != nil {
		updateVolMetrics(lines, s, streamUrl)
	}
}

func updateVolMetrics(lines []string, s DefaultStreamVolDetectService, streamUrl string) {
	for _, line := range lines {
		if strings.Contains(line, "mean_volume") {
			re := regexp.MustCompile(`[-]\d*[\.]\d`)
			allNums := re.FindAllString(line, -1)
			for _, num := range allNums {
				f, err := strconv.ParseFloat(num, 64)
				if err == nil {
					delta := s.Cfg.RunTime.StreamVolumes[streamUrl] - f
					if delta == 0.0 {
						deltaZeroCount++
					} else {
						deltaZeroCount = 0
					}
					if deltaZeroCount > 3 {
						logger.Warn(fmt.Sprintf("Volume has remained the same for %v cycles!", deltaZeroCount))
					}
					logger.Info(fmt.Sprintf("Delta: %.2f", delta))
					s.Cfg.RunTime.StreamVolumes[streamUrl] = f
					s.Cfg.Metrics.StreamVolume.WithLabelValues(streamUrl).Set(f)
				}
			}
		}
	}
}

func runFfmpeg(s DefaultStreamVolDetectService, streamUrl string) []string {
	ctx := context.Background()
	timeout := time.Duration(s.Cfg.StreamVolDetect.Duration+5) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	cmd := exec.CommandContext(ctx, s.Cfg.StreamVolDetect.FfmpegExe, "-t", strconv.Itoa(s.Cfg.StreamVolDetect.Duration), "-i", streamUrl, "-af", "volumedetect", "-f", "null", "/dev/null")
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
