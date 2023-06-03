package service

import (
	"fmt"
	"net/http"
	"time"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/services_utils/api_error"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type GpioSwitchService interface {
	Switch(string) api_error.ApiErr
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

func (s DefaultGpioSwitchService) Switch(xPoint string) (err api_error.ApiErr) {
	logger.Info(fmt.Sprintf("In Switch method. xPoint: %v", xPoint))
	// http://192.168.178.46/setDO.html?Pin=23&State=T&u=reader&p=reader
	return nil
}
