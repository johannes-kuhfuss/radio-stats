package service

import (
	"fmt"
	"net/http"
	"time"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type GpioPollService interface {
	Poll()
}

type DefaultGpioPollService struct {
	Cfg *config.AppConfig
}

var (
	runPoll        bool = false
	httpGpioTr     http.Transport
	httpGpioClient http.Client
)

func NewGpioPollService(cfg *config.AppConfig) DefaultGpioPollService {
	InitGpioHttp()
	return DefaultGpioPollService{
		Cfg: cfg,
	}
}

func InitGpioHttp() {
	httpGpioTr = http.Transport{
		DisableKeepAlives:  false,
		DisableCompression: false,
		MaxIdleConns:       0,
		IdleConnTimeout:    0,
	}
	httpGpioClient = http.Client{Transport: &httpStreamTr}
}

func (s DefaultGpioPollService) Poll() {
	if s.Cfg.Gpio.Url == "" {
		logger.Warn("No GPIO poll URL given. Not polling GPIOs")
		s.Cfg.RunTime.RunPoll = false
	} else {
		logger.Info(fmt.Sprintf("Starting to poll GPIOs from %v", s.Cfg.Gpio.Url))
		s.Cfg.RunTime.RunPoll = true
	}

	for s.Cfg.RunTime.RunPoll == true {
		//PollRun(s)
		time.Sleep(time.Duration(s.Cfg.Gpio.IntervalSec) * time.Second)
	}
}

/*
func setGpioState(cfg *config.AppConfig, stateBool []bool) {
	cfg.RunTime.Gpio01State = !stateBool[7]
	cfg.RunTime.Gpio02State = !stateBool[6]
	cfg.RunTime.Gpio03State = !stateBool[5]
	cfg.RunTime.Gpio04State = !stateBool[4]
	cfg.RunTime.Gpio05State = !stateBool[3]
	cfg.RunTime.Gpio06State = !stateBool[2]
	cfg.RunTime.Gpio07State = !stateBool[1]
	cfg.RunTime.Gpio08State = !stateBool[0]
}

func updateGpioMetrics(cfg *config.AppConfig, stateBool []bool) {
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio01Name).Set(float64(boolToInt(!stateBool[7])))
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio02Name).Set(float64(boolToInt(!stateBool[6])))
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio03Name).Set(float64(boolToInt(!stateBool[5])))
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio04Name).Set(float64(boolToInt(!stateBool[4])))
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio05Name).Set(float64(boolToInt(!stateBool[3])))
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio06Name).Set(float64(boolToInt(!stateBool[2])))
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio07Name).Set(float64(boolToInt(!stateBool[1])))
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio08Name).Set(float64(boolToInt(!stateBool[0])))
}

func boolToInt(state bool) int {
	if state == true {
		return 1
	} else {
		return 0
	}
}
*/
