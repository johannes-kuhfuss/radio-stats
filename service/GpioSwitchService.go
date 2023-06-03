package service

import (
	"net/http"
	"time"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type GpioSwitchService interface {
	Switch(xPoint string) (success bool)
}

type DefaultGpioSwitchService struct {
	Cfg *config.AppConfig
}

var (
	httpGpioSwitchTr     http.Transport
	httpGpioSwitchClient http.Client
)

func NewGpioSwitchService(cfg *config.AppConfig) DefaultGpioSwitchService {
	InitGpioSwitchHttp()
	return DefaultGpioSwitchService{
		Cfg: cfg,
	}
}

func InitGpioSwitchHttp() {
	httpGpioSwitchTr = http.Transport{
		DisableKeepAlives:  false,
		DisableCompression: false,
		MaxIdleConns:       0,
		IdleConnTimeout:    0,
	}
	httpGpioSwitchClient = http.Client{
		Transport: &httpGpioPollTr,
		Timeout:   5 * time.Second,
	}
}

func (s DefaultGpioSwitchService) Switch(xPoint string) (success bool) {
	logger.Info("Init Switch Service")
	return false
}
