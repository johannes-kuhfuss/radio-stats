package service

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/stretchr/testify/assert"
)

func TestSwitchNoHostReturnsError(t *testing.T) {
	cfg := config.AppConfig{}
	svc := NewGpioSwitchService(&cfg)

	err := svc.Switch("xpoint")

	assert.NotNil(t, err)
	assert.EqualValues(t, 500, err.StatusCode())
	assert.EqualValues(t, "No GPIO host configured", err.Message())
}

func TestSwitchSendsTwoRequests(t *testing.T) {
	var gotPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.String())
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	serverUrl, _ := url.Parse(server.URL)

	cfg := config.AppConfig{}
	cfg.Gpio.Host = serverUrl.Host
	cfg.Gpio.User = "reader"
	cfg.Gpio.Password = "pw"
	cfg.Gpio.OutConfig = map[string]int{"studio": 7}
	svc := NewGpioSwitchService(&cfg)
	svc.Delay = 0

	err := svc.Switch("studio")

	assert.Nil(t, err)
	assert.EqualValues(t, 2, len(gotPaths))
	assert.EqualValues(t, "/setDO.html?Pin=7&State=T&u=reader&p=pw", gotPaths[0])
	assert.EqualValues(t, gotPaths[0], gotPaths[1])
}

func TestSwitchFirstRequestErrorReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	serverUrl, _ := url.Parse(server.URL)

	cfg := config.AppConfig{}
	cfg.Gpio.Host = serverUrl.Host
	cfg.Gpio.OutConfig = map[string]int{"studio": 7}
	svc := NewGpioSwitchService(&cfg)
	svc.Delay = 0

	err := svc.Switch("studio")

	assert.NotNil(t, err)
	assert.EqualValues(t, 500, err.StatusCode())
}

func TestSwitchSecondRequestErrorReturnsError(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	serverUrl, _ := url.Parse(server.URL)

	cfg := config.AppConfig{}
	cfg.Gpio.Host = serverUrl.Host
	cfg.Gpio.OutConfig = map[string]int{"studio": 7}
	svc := NewGpioSwitchService(&cfg)
	svc.Delay = 0

	err := svc.Switch("studio")

	assert.NotNil(t, err)
	assert.EqualValues(t, 500, err.StatusCode())
	assert.EqualValues(t, 2, requests)
}

func TestInitGpioSwitchHttpSetsTimeout(t *testing.T) {
	client := NewGpioSwitchHttpClient()

	assert.EqualValues(t, 5*time.Second, client.Timeout)
}
