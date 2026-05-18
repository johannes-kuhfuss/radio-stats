package service

import (
	"bufio"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

var (
	volCfg     config.AppConfig
	volService DefaultStreamVolDetectService
)

const (
	streamingUrl = "https://streaming.fueralle.org/coloradio_160.ogg"
)

func TestListenNoUrlSetsRunToFalse(t *testing.T) {
	volCfg = config.AppConfig{}
	volService = NewStreamVolDetectService(&volCfg)
	volService.Cfg.SetRunListen(true)

	volService.Listen()

	assert.EqualValues(t, false, volCfg.RunTime.RunListen)
}

func TestRunFfmpegErrorExecReturnsNil(t *testing.T) {
	volCfg = config.AppConfig{}
	volService = NewStreamVolDetectService(&volCfg)
	volService.FfmpegRunner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, errors.New("ffmpeg failed")
	}

	result := volService.runFfmpeg("")

	assert.Nil(t, result)
}

func TestRunFfmpegLocalExecReturnsResult(t *testing.T) {
	volCfg = config.AppConfig{}
	volService = NewStreamVolDetectService(&volCfg)
	volService.FfmpegRunner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("ffmpeg version test\n[Parsed_volumedetect_0] mean_volume: -0.3 dB"), nil
	}
	volService.Cfg.StreamVolDetect.Urls = []string{streamingUrl}
	volService.Cfg.StreamVolDetect.Duration = 1

	result := volService.runFfmpeg(volService.Cfg.StreamVolDetect.Urls[0])

	assert.NotNil(t, result)
	assert.Contains(t, result[0], "ffmpeg version")
}

func TestUpdateMetricsUpdatesMetrics(t *testing.T) {
	var lines []string
	volCfg = config.AppConfig{}
	volService = NewStreamVolDetectService(&volCfg)
	f, _ := os.Open("../samples/ffmpeg_sample_result.txt")
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	volService.Cfg.RunTime.StreamVolumes.Vols = make(map[string]float64)
	volService.Cfg.Metrics.StreamVolume = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "volume",
		Help:      "volume detected in dB",
	}, []string{
		"streamName",
	})

	volService.updateVolMetrics(lines, streamingUrl)
	assert.EqualValues(t, -0.3, volService.Cfg.RunTime.StreamVolumes.Vols[streamingUrl])
}

func TestListenRunUpdateCounts(t *testing.T) {
	volCfg = config.AppConfig{}
	volService = NewStreamVolDetectService(&volCfg)
	volService.FfmpegRunner = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("no volume data"), nil
	}
	volService.Cfg.Metrics.StreamVolDetectCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "volume_detection_count",
		Help:      "Number of times volume level was detected on stream",
	})

	volService.ListenRun("")

	assert.EqualValues(t, 1, volService.Cfg.RunTime.StreamVolDetectCount)
}
