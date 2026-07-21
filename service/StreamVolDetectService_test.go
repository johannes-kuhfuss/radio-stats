package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const streamingURL = "https://streaming.fueralle.org/coloradio_160.ogg"

func newVolumeTestService() DefaultStreamVolDetectService {
	cfg := config.AppConfig{}
	cfg.StreamVolDetect.Duration = 1
	cfg.StreamVolDetect.IntervalSec = 1
	cfg.StreamVolDetect.FfmpegExe = "ffmpeg"
	cfg.RunTime.StreamVolumes.Vols = make(map[string]float64)
	cfg.Metrics.StreamVolDetectCount = prometheus.NewCounter(prometheus.CounterOpts{Name: "test_volume_detection_count"})
	cfg.Metrics.StreamVolume = *prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "test_volume"}, []string{"streamName"})
	return NewStreamVolDetectService(&cfg)
}

func TestListenNoURLSetsRunToFalse(t *testing.T) {
	service := newVolumeTestService()
	service.Cfg.SetRunListen(true)

	service.Listen()

	assert.False(t, service.Cfg.ShouldRunListen())
}

func TestListenStartsOnePersistentRunnerPerStream(t *testing.T) {
	service := newVolumeTestService()
	service.Cfg.StreamVolDetect.Urls = []string{"stream-one", "stream-two"}
	var starts atomic.Int32
	started := make(chan struct{}, 2)
	service.FfmpegRunner = func(ctx context.Context, _ string, _ []string, _ func(string)) error {
		starts.Add(1)
		started <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		service.ListenContext(ctx)
		close(done)
	}()

	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("ffmpeg runner did not start")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("volume listener did not stop after cancellation")
	}
	assert.EqualValues(t, 2, starts.Load())
	assert.False(t, service.Cfg.ShouldRunListen())
}

func TestListenStreamPublishesContinuousMeasurements(t *testing.T) {
	service := newVolumeTestService()
	service.Cfg.StreamVolDetect.Duration = 2
	service.Cfg.StreamVolDetect.IntervalSec = 3
	ctx, cancel := context.WithCancel(context.Background())
	service.FfmpegRunner = func(_ context.Context, _ string, args []string, onLine func(string)) error {
		assert.NotContains(t, args, "-t")
		onLine("lavfi.astats.Overall.RMS_level=-10.0")
		onLine("lavfi.astats.Overall.RMS_level=-20.0")
		cancel()
		return nil
	}

	service.listenStream(ctx, streamingURL)

	service.Cfg.RunTime.StreamVolumes.Lock()
	level := service.Cfg.RunTime.StreamVolumes.Vols[streamingURL]
	service.Cfg.RunTime.StreamVolumes.Unlock()
	assert.InDelta(t, -12.596, level, 0.001)
	assert.EqualValues(t, 1, service.Cfg.RunTime.StreamVolDetectCount)
}

func TestVolumeWindowHonorsInterval(t *testing.T) {
	window := newVolumeWindow(2, 3)

	_, complete := window.add(-10)
	assert.False(t, complete)
	level, complete := window.add(-20)
	assert.True(t, complete)
	assert.InDelta(t, -12.596, level, 0.001)

	_, complete = window.add(-30) // one-second gap before the next window
	assert.False(t, complete)
	_, complete = window.add(-30)
	assert.False(t, complete)
	level, complete = window.add(-30)
	assert.True(t, complete)
	assert.InDelta(t, -30, level, 0.001)
}

func TestParseRMSLevel(t *testing.T) {
	level, ok := parseRMSLevel("lavfi.astats.Overall.RMS_level=-14.4")
	require.True(t, ok)
	assert.Equal(t, -14.4, level)

	level, ok = parseRMSLevel("lavfi.astats.Overall.RMS_level=-inf")
	assert.True(t, ok)
	assert.True(t, math.IsInf(level, -1))
	_, ok = parseRMSLevel("unrelated ffmpeg output")
	assert.False(t, ok)
}

func TestVolumeWindowPublishesDigitalSilence(t *testing.T) {
	window := newVolumeWindow(2, 2)
	window.add(math.Inf(-1))
	level, complete := window.add(math.Inf(-1))
	require.True(t, complete)
	assert.True(t, math.IsInf(level, -1))
}

func TestFfmpegArgsUseContinuousAstats(t *testing.T) {
	args := ffmpegArgs(streamingURL)
	joined := strings.Join(args, " ")

	assert.NotContains(t, args, "-t")
	assert.Contains(t, joined, "astats=metadata=1:reset=1")
	assert.Contains(t, joined, "ametadata=mode=print")
	assert.Contains(t, joined, "asetnsamples=n=48000")
	assert.Contains(t, joined, "-reconnect_streamed 1")
}

func TestScanLinesReturnsScannerError(t *testing.T) {
	reader := &errorReader{}
	err := scanLines(reader, func(string) {})
	assert.Error(t, err)
}

type errorReader struct{}

func (*errorReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestPowerAverageIsNotDBAverage(t *testing.T) {
	window := newVolumeWindow(2, 2)
	window.add(-10)
	level, complete := window.add(-20)
	require.True(t, complete)
	assert.False(t, math.Abs(level-(-15)) < 0.001)
}
