// package app ties together all bits and pieces to start the program
package app

import (
	"github.com/johannes-kuhfuss/services_utils/logger"
	"github.com/prometheus/client_golang/prometheus"
)

var metricsRegisterer prometheus.Registerer = prometheus.DefaultRegisterer

// initMetrics sets up the Prometheus metrics
func initMetrics() {
	initMetricsWithRegisterer(metricsRegisterer)
}

func initMetricsWithRegisterer(registerer prometheus.Registerer) {
	streamListenerGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "listener_count",
		Help:      "Number of listeners per stream",
	}, []string{
		"streamName",
	})
	cfg.Metrics.StreamListenerGauge = *registerGaugeVec(registerer, streamListenerGauge)

	streamScrapeCount := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "scrape_count",
		Help:      "Number of times stream count data was retrieved from streaming server",
	})
	cfg.Metrics.StreamScrapeCount = registerCounter(registerer, streamScrapeCount)

	gpioStateGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "GPIOs",
		Name:      "status",
		Help:      "Status of GPIO 1 (active) or 0 (inactive)",
	}, []string{
		"gpioName",
	})
	cfg.Metrics.GpioStateGauge = *registerGaugeVec(registerer, gpioStateGauge)

	streamVolDetectCount := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "volume_detection_count",
		Help:      "Number of times volume level was detected on stream",
	})
	cfg.Metrics.StreamVolDetectCount = registerCounter(registerer, streamVolDetectCount)

	streamVolume := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "volume",
		Help:      "volume detected in dB",
	}, []string{
		"streamName",
	})
	cfg.Metrics.StreamVolume = *registerGaugeVec(registerer, streamVolume)

	streamAudioPeak := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "audio_peak_dbfs",
		Help:      "Peak audio sample level in dBFS over the latest one-second window",
	}, []string{"streamName"})
	cfg.Metrics.StreamAudioPeak = *registerGaugeVec(registerer, streamAudioPeak)

	streamAudioLoudness := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "audio_loudness_shortterm_lufs",
		Help:      "EBU R128 short-term audio loudness in LUFS",
	}, []string{"streamName"})
	cfg.Metrics.StreamAudioLoudness = *registerGaugeVec(registerer, streamAudioLoudness)

	streamAudioSilent := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "audio_silent",
		Help:      "Whether audio has remained below the configured silence threshold for the configured duration",
	}, []string{"streamName"})
	cfg.Metrics.StreamAudioSilent = *registerGaugeVec(registerer, streamAudioSilent)

	streamSilenceDuration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "audio_silence_duration_seconds",
		Help:      "Current consecutive time that audio has remained below the configured silence threshold",
	}, []string{"streamName"})
	cfg.Metrics.StreamSilenceDuration = *registerGaugeVec(registerer, streamSilenceDuration)

	streamVolDetectorUp := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "volume_detector_up",
		Help:      "Whether ffmpeg has produced valid measurements since it started and has not exited",
	}, []string{"streamName"})
	cfg.Metrics.StreamVolDetectorUp = *registerGaugeVec(registerer, streamVolDetectorUp)

	streamVolRestarts := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "volume_detector_restarts_total",
		Help:      "Number of times the ffmpeg volume detector process was restarted",
	}, []string{"streamName"})
	cfg.Metrics.StreamVolRestarts = *registerCounterVec(registerer, streamVolRestarts)

	streamVolLastSample := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "volume_last_measurement_timestamp_seconds",
		Help:      "Unix timestamp of the latest valid volume detector measurement",
	}, []string{"streamName"})
	cfg.Metrics.StreamVolLastSample = *registerGaugeVec(registerer, streamVolLastSample)
}

func registerGaugeVec(registerer prometheus.Registerer, collector *prometheus.GaugeVec) *prometheus.GaugeVec {
	if err := registerer.Register(collector); err != nil {
		if alreadyRegistered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.GaugeVec); ok {
				return existing
			}
		}
		logger.Error("Could not register Prometheus gauge", err)
	}
	return collector
}

func registerCounter(registerer prometheus.Registerer, collector prometheus.Counter) prometheus.Counter {
	if err := registerer.Register(collector); err != nil {
		if alreadyRegistered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := alreadyRegistered.ExistingCollector.(prometheus.Counter); ok {
				return existing
			}
		}
		logger.Error("Could not register Prometheus counter", err)
	}
	return collector
}

func registerCounterVec(registerer prometheus.Registerer, collector *prometheus.CounterVec) *prometheus.CounterVec {
	if err := registerer.Register(collector); err != nil {
		if alreadyRegistered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.CounterVec); ok {
				return existing
			}
		}
		logger.Error("Could not register Prometheus counter vector", err)
	}
	return collector
}
