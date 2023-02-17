package service

import (
	"fmt"
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

func NewGpioPollService(cfg *config.AppConfig) DefaultGpioPollService {
	return DefaultGpioPollService{
		Cfg: cfg,
	}
}

func (s DefaultGpioPollService) Poll() {
	logger.Info(fmt.Sprintf("Starting to poll GPIOs on port %v", s.Cfg.Gpio.SerialPort))

	for {
		time.Sleep(time.Duration(s.Cfg.Gpio.GpioPollIntervalSec) * time.Second)
	}
}
