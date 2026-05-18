// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/johannes-kuhfuss/emberplus/emberclient"
	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type EmberPoller interface {
	Poll()
	PollContext(context.Context)
}

type EmberClientFactory func(string, int) (config.EmberConnection, error)

type DefaultEmberPollService struct {
	Cfg           *config.AppConfig
	ClientFactory EmberClientFactory
}

func NewEmberPollService(cfg *config.AppConfig) DefaultEmberPollService {
	return DefaultEmberPollService{
		Cfg:           cfg,
		ClientFactory: newEmberClient,
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
		emberClientConfig := config.EmberConfig{
			Port:          hostData.Port,
			EntryPath:     hostData.EntryPath,
			MetricsPrefix: hostData.MetricsPrefix,
			GPIOs:         hostData.GPIOs,
		}
		emberClient, err := s.ClientFactory(host, emberClientConfig.Port)
		if err != nil {
			logger.Error(fmt.Sprintf("could not create Ember connection to host %v on port %v", host, emberClientConfig.Port), err)
			continue
		}
		emberClientConfig.Conn = emberClient
		if err := emberClientConfig.Conn.Connect(); err != nil {
			logger.Error(fmt.Sprintf("could not connect to Ember host %v on port %v", host, emberClientConfig.Port), err)
			continue
		}
		s.Cfg.RunTime.Lock()
		s.Cfg.RunTime.EmberGpios[host] = emberClientConfig
		s.Cfg.RunTime.Unlock()
	}
}

func (s DefaultEmberPollService) Reconnect() {
	s.Cfg.RunTime.RLock()
	emberGpios := make(map[string]config.EmberConfig, len(s.Cfg.RunTime.EmberGpios))
	for host, hostData := range s.Cfg.RunTime.EmberGpios {
		emberGpios[host] = hostData
	}
	s.Cfg.RunTime.RUnlock()
	for host, hostData := range emberGpios {
		if hostData.Conn == nil {
			continue
		}
		hostData.Conn.Disconnect()
		if err := hostData.Conn.Connect(); err != nil {
			logger.Errorf("Could not reconnect to host %v. %v", host, err)
		}
	}
}

func (s DefaultEmberPollService) CloseEmberConn() {
	s.Cfg.RunTime.Lock()
	defer s.Cfg.RunTime.Unlock()
	for host, clientConfig := range s.Cfg.RunTime.EmberGpios {
		if clientConfig.Conn != nil {
			clientConfig.Conn.Disconnect()
		}
		delete(s.Cfg.RunTime.EmberGpios, host)
	}
}

func (s DefaultEmberPollService) Poll() {
	s.PollContext(context.Background())
}

func (s DefaultEmberPollService) PollContext(ctx context.Context) {
	if len(s.Cfg.Ember.InConfig) == 0 {
		logger.Warn("No Ember poll host(s) given. Not polling Ember")
		s.Cfg.SetRunEmberPoll(false)
	} else {
		logger.Info("Starting to poll Ember data")
		s.InitEmberConn()
		s.Cfg.SetRunEmberPoll(true)
	}
	ticker := time.NewTicker(intervalSeconds(s.Cfg.Ember.IntervalSec))
	defer ticker.Stop()
	for s.Cfg.ShouldRunEmberPoll() {
		select {
		case <-ctx.Done():
			s.Cfg.SetRunEmberPoll(false)
			s.CloseEmberConn()
			return
		default:
		}
		s.PollRun()
		select {
		case <-ctx.Done():
			s.Cfg.SetRunEmberPoll(false)
			s.CloseEmberConn()
			return
		case <-ticker.C:
		}
	}
	s.CloseEmberConn()
}

func (s DefaultEmberPollService) PollRun() {
	var emberData map[string]map[string]any
	s.Cfg.RunTime.RLock()
	emberGpios := make(map[string]config.EmberConfig, len(s.Cfg.RunTime.EmberGpios))
	for host, clientConfig := range s.Cfg.RunTime.EmberGpios {
		emberGpios[host] = clientConfig
	}
	s.Cfg.RunTime.RUnlock()
	for host, clientConfig := range emberGpios {
		if clientConfig.Conn == nil {
			logger.Error(fmt.Sprintf("Could not get data from Ember provider. Host: %v, Port: %v", host, clientConfig.Port), fmt.Errorf("no Ember connection"))
			continue
		}
		data, err := clientConfig.Conn.GetByType("node", clientConfig.EntryPath)
		if err != nil {
			s.Reconnect()
			logger.Error(fmt.Sprintf("Could not get data from Ember provider. Host: %v, Port: %v", host, clientConfig.Port), err)
			continue
		}
		if err := json.Unmarshal(data, &emberData); err != nil {
			logger.Error(fmt.Sprintf("Could not marshall data from Ember provider. Host: %v", host), err)
			continue
		}
		s.updateMetrics(clientConfig, emberData)
	}
}

func (s DefaultEmberPollService) updateMetrics(clientConfig config.EmberConfig, emberData map[string]map[string]any) {
	for e, d := range emberData {
		if slices.Contains(clientConfig.GPIOs, e) {
			if d["description"] != nil && d["value"] != nil {
				description, ok := d["description"].(string)
				if !ok {
					logger.Warn(fmt.Sprintf("Skipping Ember GPIO %v with non-string description", e))
					continue
				}
				metricsValue, ok := d["value"].(bool)
				if !ok {
					logger.Warn(fmt.Sprintf("Skipping Ember GPIO %v with non-bool value", e))
					continue
				}
				metricName := clientConfig.MetricsPrefix + description
				s.Cfg.Metrics.GpioStateGauge.WithLabelValues(metricName).Set(float64(boolToInt(metricsValue)))
			}
		}
	}
}
