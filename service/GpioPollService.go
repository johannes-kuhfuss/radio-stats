// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/radio-stats/domain"
	"github.com/johannes-kuhfuss/services_utils/logger"
	"golang.org/x/net/html/charset"
)

type GpioPoller interface {
	Poll()
}

type DefaultGpioPollService struct {
	Cfg *config.AppConfig
}

var (
	httpGpioPollTr     http.Transport
	httpGpioPollClient http.Client
	cookie             *http.Cookie
	loggedIn           bool = false
)

func NewGpioPollService(cfg *config.AppConfig) DefaultGpioPollService {
	InitGpioPollHttp()
	return DefaultGpioPollService{
		Cfg: cfg,
	}
}

func InitGpioPollHttp() {
	httpGpioPollTr = http.Transport{
		DisableKeepAlives:  false,
		DisableCompression: false,
		MaxIdleConns:       0,
		IdleConnTimeout:    0,
	}
	httpGpioPollClient = http.Client{
		Transport: &httpGpioPollTr,
		Timeout:   5 * time.Second,
	}
}

func LoginToGpio(cfg *config.AppConfig) (success bool) {
	loginString := fmt.Sprintf("u=%s&p=%s", cfg.Gpio.User, cfg.Gpio.Password)
	bodyReader := bytes.NewBuffer([]byte(loginString))
	loginUrl := url.URL{
		Scheme: "http",
		Host:   cfg.Gpio.Host,
		Path:   "/login.html",
	}
	req, _ := http.NewRequest("POST", loginUrl.String(), bodyReader)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpGpioPollClient.Do(req)
	if err != nil {
		logger.Error(fmt.Sprintf("Could not authenticate to host %v", cfg.Gpio.Host), err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		err := fmt.Errorf("host returned status %v", resp.Status)
		logger.Error(fmt.Sprintf("Could not authenticate to host %v", cfg.Gpio.Host), err)
		return false
	}
	if len(resp.Cookies()) > 0 {
		logger.Info(fmt.Sprintf("Successfully authenticated to host %v", cfg.Gpio.Host))
		cookie = resp.Cookies()[0]
		return true
	}
	logger.Error(fmt.Sprintf("Host %v did not return cookie", cfg.Gpio.Host), err)
	return false
}

func (s DefaultGpioPollService) Poll() {
	if s.Cfg.Gpio.Host == "" {
		logger.Warn("No GPIO poll host given. Not polling GPIOs")
		s.Cfg.SetRunGpioPoll(false)
	} else {
		loggedIn = LoginToGpio(s.Cfg)
		s.Cfg.SetGpioConnected(loggedIn)
		logger.Info(fmt.Sprintf("Starting to poll GPIOs from host %v", s.Cfg.Gpio.Host))
		s.Cfg.SetRunGpioPoll(true)
	}

	for s.Cfg.ShouldRunGpioPoll() {
		if loggedIn {
			err := s.PollRun()
			if err != nil {
				if err.Error() == "expected element type <devStat> but have <html>" {
					logger.Warn("Unauthenticated. Trying to re-authenticate...")
					loggedIn = false
					s.Cfg.SetGpioConnected(loggedIn)
				}
			}
		} else {
			loggedIn = LoginToGpio(s.Cfg)
			s.Cfg.SetGpioConnected(loggedIn)
		}
		time.Sleep(time.Duration(s.Cfg.Gpio.IntervalSec) * time.Second)
	}
}

func (s DefaultGpioPollService) PollRun() error {
	pollUrl := url.URL{
		Scheme: "http",
		Host:   s.Cfg.Gpio.Host,
		Path:   "/devStat.xml",
	}
	gpioState, err := GetXmlFromPollUrl(pollUrl.String())
	if err == nil {
		mapState(gpioState, s.Cfg)
		updateGpioMetrics(s.Cfg)
		return nil
	}
	return err
}

func GetXmlFromPollUrl(pollUrl string) (*domain.DevStat, error) {
	var gpioState domain.DevStat

	req, _ := http.NewRequest("GET", pollUrl, nil)
	req.AddCookie(cookie)
	resp, err := httpGpioPollClient.Do(req)
	if err != nil {
		logger.Error("Error while polling GPIO data", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		err := errors.New("URl not found")
		logger.Error("Error while polling GPIO data", err)
		return nil, err
	}

	decoder := xml.NewDecoder(resp.Body)
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&gpioState); err != nil {
		logger.Error("Error while converting GPIO data to XML", err)
		return nil, err
	}
	return &gpioState, nil
}

func mapState(gpioState *domain.DevStat, cfg *config.AppConfig) {
	cfg.RunTime.Lock()
	defer cfg.RunTime.Unlock()
	for i1, v1 := range gpioState.In {
		for i2, v2 := range cfg.RunTime.Gpios {
			if (i1 + 1) == v2.Id {
				if v2.Invert {
					cfg.RunTime.Gpios[i2].State = !stringToBool(v1)
				} else {
					cfg.RunTime.Gpios[i2].State = stringToBool(v1)
				}
			}
		}
	}
}

func stringToBool(s string) bool {
	return s != "0"
}

func updateGpioMetrics(cfg *config.AppConfig) {
	cfg.RunTime.RLock()
	defer cfg.RunTime.RUnlock()
	for _, v := range cfg.RunTime.Gpios {
		cfg.Metrics.GpioStateGauge.WithLabelValues(v.Name).Set(float64(boolToInt(v.State)))
	}
}

func boolToInt(state bool) int {
	if state {
		return 1
	}
	return 0
}
