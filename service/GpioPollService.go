// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"bytes"
	"context"
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

var ErrGpioUnauthenticated = errors.New("gpio unauthenticated")

type GpioPoller interface {
	Poll()
	PollContext(context.Context)
}

type DefaultGpioPollService struct {
	Cfg      *config.AppConfig
	Client   *http.Client
	Cookie   *http.Cookie
	LoggedIn bool
}

func NewGpioPollService(cfg *config.AppConfig) DefaultGpioPollService {
	return DefaultGpioPollService{
		Cfg:    cfg,
		Client: NewGpioPollHttpClient(),
	}
}

func NewGpioPollHttpClient() *http.Client {
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

func InitGpioPollHttp() {
}

func LoginToGpio(cfg *config.AppConfig) (success bool) {
	service := NewGpioPollService(cfg)
	return service.Login()
}

func (s *DefaultGpioPollService) Login() (success bool) {
	if s.Client == nil {
		s.Client = NewGpioPollHttpClient()
	}
	loginString := fmt.Sprintf("u=%s&p=%s", s.Cfg.Gpio.User, s.Cfg.Gpio.Password)
	bodyReader := bytes.NewBuffer([]byte(loginString))
	loginUrl := url.URL{
		Scheme: "http",
		Host:   s.Cfg.Gpio.Host,
		Path:   "/login.html",
	}
	req, _ := http.NewRequest("POST", loginUrl.String(), bodyReader)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.Client.Do(req)
	if err != nil {
		logger.Error(fmt.Sprintf("Could not authenticate to host %v", s.Cfg.Gpio.Host), err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		err := fmt.Errorf("host returned status %v", resp.Status)
		logger.Error(fmt.Sprintf("Could not authenticate to host %v", s.Cfg.Gpio.Host), err)
		return false
	}
	if len(resp.Cookies()) > 0 {
		logger.Info(fmt.Sprintf("Successfully authenticated to host %v", s.Cfg.Gpio.Host))
		s.Cookie = resp.Cookies()[0]
		return true
	}
	logger.Error(fmt.Sprintf("Host %v did not return cookie", s.Cfg.Gpio.Host), nil)
	return false
}

func (s DefaultGpioPollService) Poll() {
	s.PollContext(context.Background())
}

func (s DefaultGpioPollService) PollContext(ctx context.Context) {
	if s.Client == nil {
		s.Client = NewGpioPollHttpClient()
	}
	if s.Cfg.Gpio.Host == "" {
		logger.Warn("No GPIO poll host given. Not polling GPIOs")
		s.Cfg.SetRunGpioPoll(false)
	} else {
		s.LoggedIn = s.Login()
		s.Cfg.SetGpioConnected(s.LoggedIn)
		logger.Info(fmt.Sprintf("Starting to poll GPIOs from host %v", s.Cfg.Gpio.Host))
		s.Cfg.SetRunGpioPoll(true)
	}

	ticker := time.NewTicker(intervalSeconds(s.Cfg.Gpio.IntervalSec))
	defer ticker.Stop()
	for s.Cfg.ShouldRunGpioPoll() {
		select {
		case <-ctx.Done():
			s.Cfg.SetRunGpioPoll(false)
			return
		default:
		}
		if s.LoggedIn {
			err := s.PollRun()
			if errors.Is(err, ErrGpioUnauthenticated) {
				logger.Warn("Unauthenticated. Trying to re-authenticate...")
				s.LoggedIn = false
				s.Cfg.SetGpioConnected(s.LoggedIn)
			}
		} else {
			s.LoggedIn = s.Login()
			s.Cfg.SetGpioConnected(s.LoggedIn)
		}
		select {
		case <-ctx.Done():
			s.Cfg.SetRunGpioPoll(false)
			return
		case <-ticker.C:
		}
	}
}

func (s DefaultGpioPollService) PollRun() error {
	pollUrl := url.URL{
		Scheme: "http",
		Host:   s.Cfg.Gpio.Host,
		Path:   "/devStat.xml",
	}
	gpioState, err := s.GetXmlFromPollUrl(pollUrl.String())
	if err == nil {
		mapState(gpioState, s.Cfg)
		updateGpioMetrics(s.Cfg)
		return nil
	}
	return err
}

func GetXmlFromPollUrl(pollUrl string) (*domain.DevStat, error) {
	return DefaultGpioPollService{Client: NewGpioPollHttpClient()}.GetXmlFromPollUrl(pollUrl)
}

func (s DefaultGpioPollService) GetXmlFromPollUrl(pollUrl string) (*domain.DevStat, error) {
	var gpioState domain.DevStat
	if s.Client == nil {
		s.Client = NewGpioPollHttpClient()
	}

	req, _ := http.NewRequest("GET", pollUrl, nil)
	if s.Cookie != nil {
		req.AddCookie(s.Cookie)
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		logger.Error("Error while polling GPIO data", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		logger.Error("Error while polling GPIO data", ErrGpioUnauthenticated)
		return nil, ErrGpioUnauthenticated
	}
	if resp.StatusCode == http.StatusNotFound {
		err := errors.New("URL not found")
		logger.Error("Error while polling GPIO data", err)
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		err := fmt.Errorf("gpio poll host returned status %v", resp.Status)
		logger.Error("Error while polling GPIO data", err)
		return nil, err
	}

	decoder := xml.NewDecoder(resp.Body)
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&gpioState); err != nil {
		if syntaxErr := new(xml.SyntaxError); errors.As(err, &syntaxErr) && syntaxErr.Msg == "expected element type <devStat> but have <html>" {
			logger.Error("Error while converting GPIO data to XML", ErrGpioUnauthenticated)
			return nil, ErrGpioUnauthenticated
		}
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
