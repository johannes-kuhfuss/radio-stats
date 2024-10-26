// package app ties together all bits and pieces to start the program
package app

import "github.com/prometheus/client_golang/prometheus"

// initMetrics sets up the Prometheus metrics
func initMetrics() {
	cfg.Metrics.StreamListenerGauge = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "listener_count",
		Help:      "Number of listeners per stream",
	}, []string{
		"streamName",
	})
	cfg.Metrics.StreamScrapeCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "scrape_count",
		Help:      "Number of times stream count data was retrieved from streaming server",
	})
	cfg.Metrics.GpioStateGauge = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "GPIOs",
		Name:      "status",
		Help:      "Status of GPIO 1 (active) or 0 (inactive)",
	}, []string{
		"gpioName",
	})
	cfg.Metrics.StreamVolDetectCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "volume_detection_count",
		Help:      "Number of times volume level was detected on stream",
	})
	cfg.Metrics.StreamVolume = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "volume",
		Help:      "volume detected in dB",
	}, []string{
		"streamName",
	})
	prometheus.MustRegister(cfg.Metrics.StreamListenerGauge)
	prometheus.MustRegister(cfg.Metrics.StreamScrapeCount)
	prometheus.MustRegister(cfg.Metrics.GpioStateGauge)
	prometheus.MustRegister(cfg.Metrics.StreamVolDetectCount)
	prometheus.MustRegister(cfg.Metrics.StreamVolume)
}
