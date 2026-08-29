// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/johannes-kuhfuss/emberplus/ember"
	"github.com/johannes-kuhfuss/emberplus/emberclient"
	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type EmberPoller interface {
	Poll()
	PollContext(context.Context)
}

type EmberClientFactory func(string, int) (config.EmberConnection, error)

type emberMetricState struct {
	description string
	value       bool
	hasValue    bool
}

type emberState struct {
	sync.Mutex
	providers map[string]map[string]emberMetricState
}

type DefaultEmberPollService struct {
	Cfg           *config.AppConfig
	ClientFactory EmberClientFactory
	state         *emberState
}

func NewEmberPollService(cfg *config.AppConfig) DefaultEmberPollService {
	return DefaultEmberPollService{
		Cfg:           cfg,
		ClientFactory: newEmberClient,
		state:         &emberState{providers: make(map[string]map[string]emberMetricState)},
	}
}

func newEmberClient(host string, port int) (config.EmberConnection, error) {
	return emberclient.NewEmberClient(host, port)
}

func (s DefaultEmberPollService) InitEmberConn() {
	if s.ClientFactory == nil {
		s.ClientFactory = newEmberClient
	}
	for host, hostData := range s.Cfg.Ember.InConfig {
		clientConfig := hostData
		client, err := s.ClientFactory(host, clientConfig.Port)
		if err != nil {
			logger.Error(fmt.Sprintf("could not create Ember connection to host %v on port %v", host, clientConfig.Port), err)
			continue
		}
		clientConfig.Conn = client
		s.Cfg.RunTime.Lock()
		s.Cfg.RunTime.EmberGpios[host] = clientConfig
		s.Cfg.RunTime.Unlock()
	}
}

func (s DefaultEmberPollService) CloseEmberConn() {
	s.Cfg.RunTime.Lock()
	defer s.Cfg.RunTime.Unlock()
	for host, clientConfig := range s.Cfg.RunTime.EmberGpios {
		if clientConfig.Conn != nil {
			_ = clientConfig.Conn.Disconnect()
		}
		delete(s.Cfg.RunTime.EmberGpios, host)
	}
}

func (s DefaultEmberPollService) Poll() {
	s.PollContext(context.Background())
}

// PollContext maintains one permanent notification reader per configured Ember
// provider. Directory discovery happens once per connection; subsequent partial
// updates are merged into the state represented by the Prometheus gauges.
func (s DefaultEmberPollService) PollContext(ctx context.Context) {
	if len(s.Cfg.Ember.InConfig) == 0 {
		logger.Warn("No Ember poll host(s) given. Not polling Ember")
		s.Cfg.SetRunEmberPoll(false)
		return
	}

	logger.Info("Starting Ember notification receivers")
	s.InitEmberConn()
	s.Cfg.SetRunEmberPoll(true)

	s.Cfg.RunTime.RLock()
	providers := make(map[string]config.EmberConfig, len(s.Cfg.RunTime.EmberGpios))
	for host, clientConfig := range s.Cfg.RunTime.EmberGpios {
		providers[host] = clientConfig
	}
	s.Cfg.RunTime.RUnlock()

	var workers sync.WaitGroup
	for host, clientConfig := range providers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			s.serveEmberProvider(ctx, host, clientConfig)
		}()
	}
	workers.Wait()
	s.Cfg.SetRunEmberPoll(false)
	s.CloseEmberConn()
}

func (s DefaultEmberPollService) serveEmberProvider(ctx context.Context, host string, clientConfig config.EmberConfig) {
	retryDelay := intervalSeconds(s.Cfg.Ember.IntervalSec)
	for ctx.Err() == nil && s.Cfg.ShouldRunEmberPoll() {
		if err := s.runEmberSession(ctx, host, clientConfig); err != nil && ctx.Err() == nil {
			logger.Error(fmt.Sprintf("Ember receiver stopped. Host: %v, Port: %v", host, clientConfig.Port), err)
		}
		_ = clientConfig.Conn.Disconnect()

		timer := time.NewTimer(retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (s DefaultEmberPollService) runEmberSession(ctx context.Context, host string, clientConfig config.EmberConfig) error {
	if clientConfig.Conn == nil {
		return fmt.Errorf("no Ember connection")
	}
	if err := clientConfig.Conn.Connect(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	elements, err := clientConfig.Conn.GetElementCollectionGlow250(ember.NodeElement, clientConfig.EntryPath)
	if err != nil {
		return fmt.Errorf("initial directory discovery: %w", err)
	}
	s.applyElements(host, clientConfig, elements)

	return clientConfig.Conn.Serve(ctx, func(message ember.RootMessage) error {
		s.applyElements(host, clientConfig, message.Elements)
		return nil
	})
}

func (s DefaultEmberPollService) applyElements(host string, clientConfig config.EmberConfig, elements ember.ElementCollection) {
	for key, element := range elements {
		s.applyElement(host, clientConfig, key.Path, element)
	}
}

func (s DefaultEmberPollService) applyElement(host string, clientConfig config.EmberConfig, path string, element *ember.Element) {
	if element == nil {
		return
	}
	if element.Path != "" {
		path = element.Path
	}

	if gpio, ok := configuredGPIO(clientConfig, path); ok {
		s.mergeMetricElement(host, clientConfig, gpio, element)
	}
	for _, child := range element.Children {
		childPath := child.Path
		if path != "" && childPath != "" && !strings.HasPrefix(childPath, path+".") {
			childPath = path + "." + childPath
		}
		s.applyElement(host, clientConfig, childPath, child)
	}
}

func configuredGPIO(clientConfig config.EmberConfig, path string) (string, bool) {
	relativePath := strings.TrimPrefix(path, clientConfig.EntryPath+".")
	for _, gpio := range clientConfig.GPIOs {
		relativeGPIO := strings.TrimPrefix(gpio, clientConfig.EntryPath+".")
		if relativePath == relativeGPIO {
			return gpio, true
		}
	}
	return "", false
}

func (s DefaultEmberPollService) mergeMetricElement(host string, clientConfig config.EmberConfig, gpio string, element *ember.Element) {
	s.state.Lock()
	provider := s.state.providers[host]
	if provider == nil {
		provider = make(map[string]emberMetricState)
		s.state.providers[host] = provider
	}
	state := provider[gpio]
	if element.Description != "" {
		state.description = element.Description
	}
	if element.HasValue {
		value, ok := element.Value.(bool)
		if !ok {
			s.state.Unlock()
			logger.Warn(fmt.Sprintf("Skipping Ember GPIO %v with non-bool value", gpio))
			return
		}
		state.value = value
		state.hasValue = true
	}
	provider[gpio] = state
	s.state.Unlock()

	if state.description != "" && state.hasValue {
		metricName := clientConfig.MetricsPrefix + state.description
		s.Cfg.Metrics.GpioStateGauge.WithLabelValues(metricName).Set(float64(boolToInt(state.value)))
	}
}
