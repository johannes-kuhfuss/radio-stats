package service

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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
	if s.Cfg.Gpio.Host == "" {
		return api_error.NewInternalServerError("No GPIO host configured", nil)
	}
	// Format expected: http://<IP or DNS name>/setDO.html?Pin=<pin number>&State=T&u=<user name>&p=<password>
	switchUrl := url.URL{
		Scheme: "http",
		Host:   s.Cfg.Gpio.Host,
		Path:   "/setDO.html",
	}
	queryOn := switchUrl.Query()
	xPointNum := strconv.Itoa(s.Cfg.Gpio.OutConfig[xPoint])
	queryOn.Set("Pin", xPointNum)
	queryOn.Set("State", "T")
	switchUrl.RawQuery = queryOn.Encode()
	// add user and password manually since device expects it in this particular order
	userString := fmt.Sprintf("&u=%v", s.Cfg.Gpio.User)
	pwString := fmt.Sprintf("&p=%v", s.Cfg.Gpio.Password)
	urlString := switchUrl.String() + userString + pwString

	// Toggle twice
	req, _ := http.NewRequest("GET", urlString, nil)
	resp, reqErr := httpGpioSwitchClient.Do(req)
	if reqErr != nil {
		msg := "Error while switching xpoint (1/2)"
		logger.Error(msg, reqErr)
		return api_error.NewInternalServerError(msg, reqErr)
	}

	time.Sleep(500 * time.Millisecond)

	resp, reqErr = httpGpioSwitchClient.Do(req)
	if reqErr != nil {
		msg := "Error while switching xpoint (2/2)"
		logger.Error(msg, reqErr)
		return api_error.NewInternalServerError(msg, reqErr)
	}

	defer resp.Body.Close()

	return nil
}
