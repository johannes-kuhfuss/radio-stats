package service

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	cookie         *http.Cookie
)

func NewGpioPollService(cfg *config.AppConfig) DefaultGpioPollService {
	InitGpioHttp()
	LoginToGpio(cfg)
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
	httpGpioClient = http.Client{Transport: &httpGpioTr}
}

func LoginToGpio(cfg *config.AppConfig) {
	loginString := fmt.Sprintf("u=%s&p=%s", cfg.Gpio.User, cfg.Gpio.Password)
	bodyReader := bytes.NewBuffer([]byte(loginString))
	loginUrl := url.URL{
		Scheme: "http",
		Host:   cfg.Gpio.Host,
		Path:   "/login.html",
	}
	req, _ := http.NewRequest("POST", loginUrl.String(), bodyReader)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpGpioClient.Do(req)
	if err != nil {
		logger.Error("Error while logging in to GPIO host", err)
	} else {
		cookie = resp.Cookies()[0]
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
		PollRun(s)
		time.Sleep(time.Duration(s.Cfg.Gpio.IntervalSec) * time.Second)
	}
}

func PollRun(s DefaultGpioPollService) {
	pollUrl := url.URL{
		Scheme: "http",
		Host:   s.Cfg.Gpio.Host,
		Path:   "/devStat.xml",
	}
	body, err := GetDataFromPollUrl(pollUrl.String())
	if err == nil {
		// process data
		logger.Debug(string(body))
	}
}

func GetDataFromPollUrl(pollUrl string) ([]byte, error) {
	req, _ := http.NewRequest("GET", pollUrl, nil)
	req.AddCookie(cookie)
	resp, err := httpGpioClient.Do(req)
	if err != nil {
		logger.Error("Error while polling GPIO data", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error while reading GPIO data", err)
		return nil, err
	}
	return body, nil
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
