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

func Test_NewStreamVolDetectService_ReturnsDefaultStreamVolDetectService(t *testing.T) {
	volService = NewStreamVolDetectService(&volCfg)

	assert.EqualValues(t, volCfg, *volService.Cfg)
}

func Test_Listen_NoUrl_SetsRunToFalse(t *testing.T) {
	volService = NewStreamVolDetectService(&volCfg)
	volService.Cfg.StreamScrape.Url = ""

	volService.Listen()

	assert.EqualValues(t, false, volCfg.RunTime.RunScrape)
}

func Test_runFfmpeg_ErrorExec_ReturnsNil(t *testing.T) {
	volService = NewStreamVolDetectService(&volCfg)
	volService.Cfg.StreamVolDetect.FfmpegExe = "i-dont-exist"

	result := runFfmpeg(volService)

	assert.Nil(t, result)
}

func Test_runFfmpeg_LocalExec_ReturnsResult(t *testing.T) {
	volService = NewStreamVolDetectService(&volCfg)
	volService.Cfg.StreamVolDetect.FfmpegExe = "../prog/ffmpeg.exe"
	volService.Cfg.StreamVolDetect.Url = "https://streaming.fueralle.org/coloradio_48.aac"
	volService.Cfg.StreamVolDetect.Duration = 2

	result := runFfmpeg(volService)

	assert.NotNil(t, result)
	assert.Contains(t, result[0], "ffmpeg version")
}

func Test_updateMetrics_UpdatesMetrics(t *testing.T) {
	var lines []string
	volService = NewStreamVolDetectService(&volCfg)
	f, _ := os.Open("ffmpeg_sample_result.txt")
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	volService.Cfg.Metrics.StreamVolume = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "volume",
		Help:      "volume detected in dB",
	}, []string{
		"streamName",
	})

	updateVolMetrics(lines, volService)
	assert.EqualValues(t, -0.3, volService.Cfg.RunTime.StreamVolume)
}

func Test_ListenRun_UpdateCounts(t *testing.T) {
	volService = NewStreamVolDetectService(&volCfg)
	volService.Cfg.Metrics.StreamVolDetectCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "volume_detection_count",
		Help:      "Number of times volume level was detected on stream",
	})

	ListenRun(volService)

	assert.EqualValues(t, 1, volService.Cfg.RunTime.StreamVolDetectCount)
}