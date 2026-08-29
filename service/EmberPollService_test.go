package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/johannes-kuhfuss/emberplus/ember"
	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeEmberConn struct {
	mu              sync.Mutex
	elements        ember.ElementCollection
	getErr          error
	connectErr      error
	serveErr        error
	notifications   []ember.RootMessage
	connectCount    int
	disconnectCount int
	getCount        int
	serveCount      int
	requestedType   ember.ElementType
	requestedPath   string
}

func (f *fakeEmberConn) Connect() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.connectCount++
	return f.connectErr
}

func (f *fakeEmberConn) Disconnect() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.disconnectCount++
	return nil
}

func (f *fakeEmberConn) GetElementCollectionGlow250(elementType ember.ElementType, path string) (ember.ElementCollection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCount++
	f.requestedType = elementType
	f.requestedPath = path
	return f.elements, f.getErr
}

func (f *fakeEmberConn) Serve(ctx context.Context, handler func(ember.RootMessage) error) error {
	f.mu.Lock()
	f.serveCount++
	notifications := append([]ember.RootMessage(nil), f.notifications...)
	serveErr := f.serveErr
	f.mu.Unlock()
	for _, notification := range notifications {
		if err := handler(notification); err != nil {
			return err
		}
	}
	if serveErr != nil {
		return serveErr
	}
	<-ctx.Done()
	return ctx.Err()
}

func newEmberTestConfig() *config.AppConfig {
	cfg := &config.AppConfig{}
	cfg.RunTime.EmberGpios = make(map[string]config.EmberConfig)
	cfg.Metrics.GpioStateGauge = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "GPIOs",
		Name:      "status",
		Help:      "Status of GPIO 1 (active) or 0 (inactive)",
	}, []string{"gpioName"})
	return cfg
}

func TestNewEmberPollServiceSetsConfig(t *testing.T) {
	cfg := config.AppConfig{}

	svc := NewEmberPollService(&cfg)

	assert.Same(t, &cfg, svc.Cfg)
	assert.NotNil(t, svc.state)
}

func TestEmberPollNoConfigSetsRunFalse(t *testing.T) {
	cfg := config.AppConfig{}
	cfg.SetRunEmberPoll(true)
	svc := NewEmberPollService(&cfg)

	svc.Poll()

	assert.False(t, cfg.ShouldRunEmberPoll())
}

func TestInitEmberConnUsesFactoryAndStoresUnconnectedClient(t *testing.T) {
	cfg := newEmberTestConfig()
	cfg.Ember.InConfig = config.EmberConfigDecoder{
		"host": {Port: 9000, EntryPath: "1.2.3", MetricsPrefix: "ember_", GPIOs: []string{"1"}},
	}
	fakeConn := &fakeEmberConn{}
	svc := NewEmberPollService(cfg)
	svc.ClientFactory = func(host string, port int) (config.EmberConnection, error) {
		assert.Equal(t, "host", host)
		assert.Equal(t, 9000, port)
		return fakeConn, nil
	}

	svc.InitEmberConn()

	require.Len(t, cfg.RunTime.EmberGpios, 1)
	assert.Same(t, fakeConn, cfg.RunTime.EmberGpios["host"].Conn)
	assert.Zero(t, fakeConn.connectCount)
}

func TestRunEmberSessionDiscoversOnceThenAppliesNotifications(t *testing.T) {
	cfg := newEmberTestConfig()
	initial := ember.NewElementCollection()
	initial[ember.ElementKey{Path: "1"}] = &ember.Element{
		Path: "1", Description: "on_air", HasValue: true, Value: false,
	}
	update := ember.NewElementCollection()
	update[ember.ElementKey{Path: "1.2.3.1"}] = &ember.Element{
		Path: "1.2.3.1", HasValue: true, Value: true,
	}
	serveStopped := errors.New("serve stopped")
	fakeConn := &fakeEmberConn{
		elements:      initial,
		notifications: []ember.RootMessage{{Elements: update}},
		serveErr:      serveStopped,
	}
	clientConfig := config.EmberConfig{
		EntryPath: "1.2.3", MetricsPrefix: "ember_", GPIOs: []string{"1"}, Conn: fakeConn,
	}
	svc := NewEmberPollService(cfg)

	err := svc.runEmberSession(context.Background(), "host", clientConfig)

	assert.ErrorIs(t, err, serveStopped)
	assert.Equal(t, 1, fakeConn.connectCount)
	assert.Equal(t, 1, fakeConn.getCount)
	assert.Equal(t, 1, fakeConn.serveCount)
	assert.Equal(t, ember.NodeElement, fakeConn.requestedType)
	assert.Equal(t, "1.2.3", fakeConn.requestedPath)
	assert.EqualValues(t, 1, gaugeValue(cfg.Metrics.GpioStateGauge.WithLabelValues("ember_on_air")))
}

func TestPartialUpdateRetainsInitialDescription(t *testing.T) {
	cfg := newEmberTestConfig()
	svc := NewEmberPollService(cfg)
	clientConfig := config.EmberConfig{EntryPath: "1.2.3", MetricsPrefix: "ember_", GPIOs: []string{"1"}}

	svc.applyElements("host", clientConfig, ember.ElementCollection{
		{Path: "1"}: {Path: "1", Description: "on_air", HasValue: true, Value: false},
	})
	svc.applyElements("host", clientConfig, ember.ElementCollection{
		{Path: "1.2.3.1"}: {Path: "1.2.3.1", HasValue: true, Value: true},
	})

	assert.EqualValues(t, 1, gaugeValue(cfg.Metrics.GpioStateGauge.WithLabelValues("ember_on_air")))
}

func TestFullyQualifiedConfiguredGPIOUpdatesMetric(t *testing.T) {
	cfg := newEmberTestConfig()
	svc := NewEmberPollService(cfg)
	clientConfig := config.EmberConfig{
		EntryPath: "0.2", MetricsPrefix: "ember_", GPIOs: []string{"0.2.0"},
	}

	svc.applyElements("host", clientConfig, ember.ElementCollection{
		{Path: "0.2.0"}: {Path: "0.2.0", Description: "on_air", HasValue: true, Value: true},
	})

	assert.EqualValues(t, 1, gaugeValue(cfg.Metrics.GpioStateGauge.WithLabelValues("ember_on_air")))
}

func TestUnconfiguredAndNonBooleanElementsAreIgnored(t *testing.T) {
	cfg := newEmberTestConfig()
	svc := NewEmberPollService(cfg)
	clientConfig := config.EmberConfig{MetricsPrefix: "ember_", GPIOs: []string{"1"}}

	svc.applyElements("host", clientConfig, ember.ElementCollection{
		{Path: "1"}: {Path: "1", Description: "invalid", HasValue: true, Value: int64(1)},
		{Path: "2"}: {Path: "2", Description: "ignored", HasValue: true, Value: true},
	})

	metrics := make(chan prometheus.Metric, 1)
	cfg.Metrics.GpioStateGauge.Collect(metrics)
	close(metrics)
	assert.Empty(t, metrics)
}

func gaugeValue(metric prometheus.Metric) float64 {
	var pb dto.Metric
	_ = metric.Write(&pb)
	return pb.GetGauge().GetValue()
}
