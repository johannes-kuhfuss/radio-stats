package service

import (
	"bufio"
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
	streamingUrl = "https://streaming.fueralle.org/coloradio_56.mp3"
)

func TestListenNoUrlSetsRunToFalse(t *testing.T) {
	volService = NewStreamVolDetectService(&volCfg)
	volService.Cfg.StreamScrape.Url = ""

	volService.Listen()

	assert.EqualValues(t, false, volCfg.RunTime.RunScrape)
}

func TestRunFfmpegErrorExecReturnsNil(t *testing.T) {
	volService = NewStreamVolDetectService(&volCfg)
	volService.Cfg.StreamVolDetect.FfmpegExe = "i-dont-exist"

	result := volService.runFfmpeg("")

	assert.Nil(t, result)
}

func TestRunFfmpegLocalExecReturnsResult(t *testing.T) {
	volService = NewStreamVolDetectService(&volCfg)
	volService.Cfg.StreamVolDetect.FfmpegExe = "../prog/ffmpeg.exe"
	volService.Cfg.StreamVolDetect.Urls = []string{streamingUrl}
	volService.Cfg.StreamVolDetect.Duration = 2

	result := volService.runFfmpeg(volService.Cfg.StreamVolDetect.Urls[0])

	assert.NotNil(t, result)
	assert.Contains(t, result[0], "ffmpeg version")
}

func TestUpdateMetricsUpdatesMetrics(t *testing.T) {
	var lines []string
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
	volService = NewStreamVolDetectService(&volCfg)
	volService.Cfg.Metrics.StreamVolDetectCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "volume_detection_count",
		Help:      "Number of times volume level was detected on stream",
	})

	volService.ListenRun("")

	assert.EqualValues(t, 1, volService.Cfg.RunTime.StreamVolDetectCount)
}
