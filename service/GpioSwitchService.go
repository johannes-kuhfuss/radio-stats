// package service implements the services and their business logic that provide the main part of the program
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

type GpioSwitcher interface {
	Switch(string) api_error.ApiErr
}

type DefaultGpioSwitchService struct {
	Cfg    *config.AppConfig
	Client *http.Client
	Delay  time.Duration
}

func NewGpioSwitchService(cfg *config.AppConfig) DefaultGpioSwitchService {
	return DefaultGpioSwitchService{
		Cfg:    cfg,
		Client: NewGpioSwitchHttpClient(),
		Delay:  250 * time.Millisecond,
	}
}

func NewGpioSwitchHttpClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:  false,
			DisableCompression: false,
			MaxIdleConns:       0,
			IdleConnTimeout:    0,
		},
		Timeout: 5 * time.Second,
	}
}

func InitGpioSwitchHttp() {
}

func (s DefaultGpioSwitchService) Switch(xPoint string) (err api_error.ApiErr) {
	if s.Client == nil {
		s.Client = NewGpioSwitchHttpClient()
	}
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

	logger.Info(fmt.Sprintf("Switching xpoint %v, pin %v", xPoint, xPointNum))

	req, reqErr := http.NewRequest("GET", urlString, nil)
	if reqErr != nil {
		msg := "Error while creating switch request"
		logger.Error(msg, reqErr)
		return api_error.NewInternalServerError(msg, reqErr)
	}

	if err := s.doSwitchRequest(req, "1/2"); err != nil {
		return err
	}

	time.Sleep(s.Delay)

	if err := s.doSwitchRequest(req, "2/2"); err != nil {
		return err
	}

	time.Sleep(s.Delay)

	return nil
}

func (s DefaultGpioSwitchService) doSwitchRequest(req *http.Request, attempt string) api_error.ApiErr {
	resp, reqErr := s.Client.Do(req)
	if reqErr != nil {
		msg := fmt.Sprintf("Error while switching xpoint (%s)", attempt)
		logger.Error(msg, reqErr)
		return api_error.NewInternalServerError(msg, reqErr)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		msg := fmt.Sprintf("Error while switching xpoint (%s), device returned status %v", attempt, resp.Status)
		logger.Error(msg, nil)
		return api_error.NewInternalServerError(msg, nil)
	}
	return nil
}
