package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/radio-stats/dto"
	"github.com/johannes-kuhfuss/radio-stats/service"
	"github.com/stretchr/testify/assert"
)

var (
	sh  GpioSwitchHandler
	svc service.DefaultGpioSwitchService
)

const (
	switchUrl = "/switch"
)

func setupSwitchUiTest() func() {
	svc = service.NewGpioSwitchService(&cfg)
	sh = NewGpioSwitchHandler(&cfg, svc)
	router = gin.Default()
	recorder = httptest.NewRecorder()
	return func() {
		router = nil
	}
}

func TestValidateReqInvalidXpointReturnsError(t *testing.T) {
	tearDown := setupSwitchUiTest()
	defer tearDown()
	swreq := dto.GpioSwitchRequest{
		Xpoint: "noexist",
	}
	err := sh.validateReq(swreq)
	assert.NotNil(t, err)
	assert.EqualValues(t, "xpoint with name noexist does not exist", err.Message())
	assert.EqualValues(t, 400, err.StatusCode())
}

func TestValidateReqValidXpointReturnsNoError(t *testing.T) {
	tearDown := setupSwitchUiTest()
	defer tearDown()
	outList := make(map[string]int)
	outList["exists"] = 1
	cfg.Gpio.OutConfig = outList
	swreq := dto.GpioSwitchRequest{
		Xpoint: "exists",
	}
	err := sh.validateReq(swreq)
	assert.Nil(t, err)
}

func TestSwitchXpointNoXpointReturnsError(t *testing.T) {
	tearDown := setupSwitchUiTest()
	defer tearDown()
	router.POST(switchUrl, sh.SwitchXpoint)
	request, _ := http.NewRequest(http.MethodPost, switchUrl, nil)

	router.ServeHTTP(recorder, request)

	assert.EqualValues(t, http.StatusBadRequest, recorder.Code)
	result, _ := io.ReadAll(recorder.Body)
	assert.EqualValues(t, "{\"message\":\"xpoint with name  does not exist\",\"statuscode\":400,\"causes\":null}", string(result))
}

func TestSwitchXpointNoSwitchReturnsError(t *testing.T) {
	tearDown := setupSwitchUiTest()
	defer tearDown()
	outList := make(map[string]int)
	outList["exists"] = 1
	cfg.Gpio.OutConfig = outList
	router.POST(switchUrl, sh.SwitchXpoint)
	cmd := url.Values{}
	cmd.Set("xpoint", "exists")
	request, _ := http.NewRequest(http.MethodPost, switchUrl, strings.NewReader(cmd.Encode()))
	request.Header.Set("Content-type", "application/x-www-form-urlencoded")

	router.ServeHTTP(recorder, request)

	assert.EqualValues(t, http.StatusInternalServerError, recorder.Code)
	result, _ := io.ReadAll(recorder.Body)
	assert.EqualValues(t, "{\"message\":\"No GPIO host configured\",\"statuscode\":500,\"causes\":null}", string(result))
}
