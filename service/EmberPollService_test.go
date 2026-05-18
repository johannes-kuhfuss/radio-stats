package service

import (
	"errors"
	"testing"

	"github.com/johannes-kuhfuss/emberplus/ember"
	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

type fakeEmberConn struct {
	data            []byte
	getErr          error
	connectErr      error
	connectCount    int
	disconnectCount int
	requestedType   ember.ElementType
	requestedPath   string
}

func (f *fakeEmberConn) Connect() error {
	f.connectCount++
	return f.connectErr
}

func (f *fakeEmberConn) Disconnect() error {
	f.disconnectCount++
	return nil
}

func (f *fakeEmberConn) GetByType(elementType ember.ElementType, path string) ([]byte, error) {
	f.requestedType = elementType
	f.requestedPath = path
	return f.data, f.getErr
}

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

func TestInitEmberConnUsesFactoryAndStoresConnection(t *testing.T) {
	cfg := config.AppConfig{}
	cfg.RunTime.EmberGpios = make(map[string]config.EmberConfig)
	cfg.Ember.InConfig = config.EmberConfigDecoder{
		"host": {Port: 9000, EntryPath: "1.2.3", MetricsPrefix: "ember_", GPIOs: []string{"1"}},
	}
	fakeConn := &fakeEmberConn{}
	svc := NewEmberPollService(&cfg)
	svc.ClientFactory = func(host string, port int) (config.EmberConnection, error) {
		assert.EqualValues(t, "host", host)
		assert.EqualValues(t, 9000, port)
		return fakeConn, nil
	}

	svc.InitEmberConn()

	assert.EqualValues(t, 1, len(cfg.RunTime.EmberGpios))
	assert.Same(t, fakeConn, cfg.RunTime.EmberGpios["host"].Conn)
	assert.EqualValues(t, 1, fakeConn.connectCount)
}

func TestPollRunReadsEmberData(t *testing.T) {
	cfg := config.AppConfig{}
	cfg.RunTime.EmberGpios = make(map[string]config.EmberConfig)
	cfg.Metrics.GpioStateGauge = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "GPIOs",
		Name:      "status",
		Help:      "Status of GPIO 1 (active) or 0 (inactive)",
	}, []string{"gpioName"})
	fakeConn := &fakeEmberConn{data: []byte(`{"1":{"description":"on_air","value":true}}`)}
	cfg.RunTime.EmberGpios["host"] = config.EmberConfig{
		EntryPath:     "1.2.3",
		MetricsPrefix: "ember_",
		GPIOs:         []string{"1"},
		Conn:          fakeConn,
	}
	svc := NewEmberPollService(&cfg)

	svc.PollRun()

	assert.EqualValues(t, ember.ElementType("node"), fakeConn.requestedType)
	assert.EqualValues(t, "1.2.3", fakeConn.requestedPath)
	assert.EqualValues(t, 1, gaugeValue(cfg.Metrics.GpioStateGauge.WithLabelValues("ember_on_air")))
}

func TestPollRunReconnectsOnReadError(t *testing.T) {
	cfg := config.AppConfig{}
	cfg.RunTime.EmberGpios = make(map[string]config.EmberConfig)
	fakeConn := &fakeEmberConn{getErr: errors.New("read failed")}
	cfg.RunTime.EmberGpios["host"] = config.EmberConfig{Conn: fakeConn}
	svc := NewEmberPollService(&cfg)

	svc.PollRun()

	assert.EqualValues(t, 1, fakeConn.disconnectCount)
	assert.EqualValues(t, 1, fakeConn.connectCount)
}

func gaugeValue(metric prometheus.Metric) float64 {
	var pb dto.Metric
	metric.Write(&pb)
	return pb.GetGauge().GetValue()
}
