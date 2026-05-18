package service

import (
	"testing"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestNewEmberPollServiceSetsConfig(t *testing.T) {
	cfg := config.AppConfig{}

	svc := NewEmberPollService(&cfg)

	assert.Same(t, &cfg, svc.Cfg)
}

func TestEmberPollNoConfigSetsRunFalse(t *testing.T) {
	cfg := config.AppConfig{}
	cfg.SetRunEmberPoll(true)
	svc := NewEmberPollService(&cfg)

	svc.Poll()

	assert.False(t, cfg.ShouldRunEmberPoll())
}

func TestEmberUpdateMetricsUpdatesConfiguredGpios(t *testing.T) {
	cfg := config.AppConfig{}
	cfg.Metrics.GpioStateGauge = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "GPIOs",
		Name:      "status",
		Help:      "Status of GPIO 1 (active) or 0 (inactive)",
	}, []string{"gpioName"})
	svc := NewEmberPollService(&cfg)
	clientConfig := config.EmberConfig{
		MetricsPrefix: "ember_",
		GPIOs:         []string{"1", "2", "3"},
	}
	emberData := map[string]map[string]any{
		"1": {"description": "on_air", "value": true},
		"2": {"description": "alarm", "value": false},
		"3": {"description": 123, "value": true},
		"4": {"description": "ignored", "value": true},
	}

	svc.updateMetrics(clientConfig, emberData)

	assert.EqualValues(t, 1, gaugeValue(cfg.Metrics.GpioStateGauge.WithLabelValues("ember_on_air")))
	assert.EqualValues(t, 0, gaugeValue(cfg.Metrics.GpioStateGauge.WithLabelValues("ember_alarm")))
}

func gaugeValue(metric prometheus.Metric) float64 {
	var pb dto.Metric
	metric.Write(&pb)
	return pb.GetGauge().GetValue()
}
