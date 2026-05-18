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
