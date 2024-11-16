// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/johannes-kuhfuss/emberplus/emberclient"
	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/services_utils/logger"
	"github.com/johannes-kuhfuss/services_utils/misc"
)

type EmberPollService interface {
	Poll()
}

type DefaultEmberPollService struct {
	Cfg *config.AppConfig
}

func NewEmberPollService(cfg *config.AppConfig) DefaultEmberPollService {
	return DefaultEmberPollService{
		Cfg: cfg,
	}
}

func (s DefaultEmberPollService) InitEmberConn() {
	for host, hostData := range s.Cfg.Ember.InConfig {
		var (
			emberClientConfig config.EmberConfig
			emberClient       *emberclient.EmberClient
		)
		emberClientConfig.Port = hostData.Port
		emberClientConfig.EntryPath = hostData.EntryPath
		emberClientConfig.MetricsPrefix = hostData.MetricsPrefix
		emberClientConfig.GPIOs = hostData.GPIOs
		emberClient, err := emberclient.NewEmberClient(host, emberClientConfig.Port)
		if err != nil {
			logger.Error(fmt.Sprintf("could not creaet Ember connection to host %v on port %v", host, emberClientConfig.Port), err)
		} else {
			emberClientConfig.Conn = emberClient
			emberClientConfig.Conn.Connect()
			s.Cfg.RunTime.EmberGpios[host] = emberClientConfig
		}
	}
}

func (s DefaultEmberPollService) Reconnect() {
	for host, hostData := range s.Cfg.RunTime.EmberGpios {
		if !hostData.Conn.IsConnected() {
			logger.Warnf("%v was disconnected. Trying to reconnec...", host)
			if err := hostData.Conn.Connect(); err != nil {
				logger.Errorf("Couldn not reconnect ot host %v. %v", host, err)
			}
		}
	}
}

func (s DefaultEmberPollService) CloseEmberConn() {
	for host, clientConfig := range s.Cfg.RunTime.EmberGpios {
		clientConfig.Conn.Disconnect()
		delete(s.Cfg.RunTime.EmberGpios, host)
	}
}

func (s DefaultEmberPollService) Poll() {
	if len(s.Cfg.Ember.InConfig) == 0 {
		logger.Warn("No Ember poll host(s) given. Not polling Ember")
		s.Cfg.RunTime.RunEmberPoll = false
	} else {
		logger.Info("Starting to poll Ember data")
		s.InitEmberConn()
		s.Cfg.RunTime.RunEmberPoll = true
	}
	for s.Cfg.RunTime.RunEmberPoll {
		s.PollRun()
		time.Sleep(time.Duration(s.Cfg.Ember.IntervalSec) * time.Second)
	}
	s.CloseEmberConn()
}

func (s DefaultEmberPollService) PollRun() {
	var emberData map[string]map[string]interface{}
	for host, clientConfig := range s.Cfg.RunTime.EmberGpios {
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

func (s DefaultEmberPollService) updateMetrics(clientConfig config.EmberConfig, emberData map[string]map[string]interface{}) {
	for e, d := range emberData {
		if misc.SliceContainsString(clientConfig.GPIOs, e) {
			if d["description"] != nil && d["value"] != nil {
				metricName := clientConfig.MetricsPrefix + d["description"].(string)
				metricsValue := d["value"].(bool)
				s.Cfg.Metrics.GpioStateGauge.WithLabelValues(metricName).Set(float64(boolToInt(metricsValue)))
			}
		}
	}
}
