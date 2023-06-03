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

type GpioPollService interface {
	Poll()
}

type DefaultGpioPollService struct {
	Cfg *config.AppConfig
}

var (
	runPoll            bool = false
	httpGpioPollTr     http.Transport
	httpGpioPollClient http.Client
	cookie             *http.Cookie
	loggedIn           bool = false
)

func NewGpioPollService(cfg *config.AppConfig) DefaultGpioPollService {
	InitGpioPollHttp()
	loggedIn = LoginToGpio(cfg)
	cfg.RunTime.GpioConnected = loggedIn
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
	} else {
		if len(resp.Cookies()) > 0 {
			logger.Info("Successfully authenticated")
			cookie = resp.Cookies()[0]
			return true
		} else {
			logger.Error(fmt.Sprintf("Host %v did not return cookie", cfg.Gpio.Host), err)
			return false
		}

	}
}

func (s DefaultGpioPollService) Poll() {
	if s.Cfg.Gpio.Host == "" {
		logger.Warn("No GPIO poll host given. Not polling GPIOs")
		s.Cfg.RunTime.RunPoll = false
	} else {
		logger.Info(fmt.Sprintf("Starting to poll GPIOs from host %v", s.Cfg.Gpio.Host))
		s.Cfg.RunTime.RunPoll = true
	}

	for s.Cfg.RunTime.RunPoll == true {
		if loggedIn {
			err := PollRun(s)
			if err != nil {
				if err.Error() == "expected element type <devStat> but have <html>" {
					logger.Warn("Unauthenticated. Trying to re-authenticate...")
					loggedIn = false
					s.Cfg.RunTime.GpioConnected = loggedIn
				}
			}
		} else {
			loggedIn = LoginToGpio(s.Cfg)
			s.Cfg.RunTime.GpioConnected = loggedIn
		}
		time.Sleep(time.Duration(s.Cfg.Gpio.IntervalSec) * time.Second)
	}
}

func PollRun(s DefaultGpioPollService) error {
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
	} else {
		return err
	}

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
	if resp.StatusCode == 404 {
		err := errors.New("URl not found")
		logger.Error("Error while polling GPIO data", err)
		return nil, err
	}
	defer resp.Body.Close()

	decoder := xml.NewDecoder(resp.Body)
	decoder.CharsetReader = charset.NewReaderLabel
	err = decoder.Decode(&gpioState)
	if err != nil {
		logger.Error("Error while converting GPIO data to XML", err)
		return nil, err
	}
	return &gpioState, nil
}

func mapState(gpioState *domain.DevStat, cfg *config.AppConfig) {
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
	if s == "0" {
		return false
	} else {
		return true
	}
}

func updateGpioMetrics(cfg *config.AppConfig) {
	for _, v := range cfg.RunTime.Gpios {
		cfg.Metrics.GpioStateGauge.WithLabelValues(v.Name).Set(float64(boolToInt(v.State)))
	}
}

func boolToInt(state bool) int {
	if state == true {
		return 1
	} else {
		return 0
	}
}
