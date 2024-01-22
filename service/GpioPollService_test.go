package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/radio-stats/domain"
	"github.com/stretchr/testify/assert"
)

var (
	gpioCfg     config.AppConfig
	gpioService DefaultGpioPollService
	server      *httptest.Server
)

func setupGpioTest(retError bool, setCookie bool, bodyData string) func() {
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expire := time.Now().AddDate(0, 0, 1)
		cookie := http.Cookie{
			Name:    "testcookie",
			Value:   "testcookie",
			Path:    "/",
			Domain:  "localhost",
			Expires: expire,
		}
		if setCookie {
			http.SetCookie(w, &cookie)
		}
		if retError {
			w.WriteHeader(404)
		} else {
			fmt.Fprint(w, bodyData)
		}

	}))
	url, _ := url.Parse(server.URL)
	gpioCfg.Gpio.Host = url.Host
	return func() {
		server.Close()
	}
}

func Test_Poll_NoHost_SetsRunPollFalse(t *testing.T) {
	teardown := setupGpioTest(false, true, "")
	defer teardown()
	gpioService = NewGpioPollService(&gpioCfg)
	gpioCfg.Gpio.Host = ""

	gpioService.Poll()

	assert.EqualValues(t, false, gpioCfg.RunTime.RunPoll)
}

func Test_LoginToGpio_CannotConnect_ReturnsFalse(t *testing.T) {
	teardown := setupGpioTest(true, false, "")
	defer teardown()

	success := LoginToGpio(&gpioCfg)

	assert.EqualValues(t, false, success)
}

func Test_LoginToGpio_NoCookie_ReturnsFalse(t *testing.T) {
	teardown := setupGpioTest(false, false, "")
	defer teardown()

	success := LoginToGpio(&gpioCfg)

	assert.EqualValues(t, false, success)
}

func Test_LoginToGpio_ReturnsTrue(t *testing.T) {
	teardown := setupGpioTest(false, true, "")
	defer teardown()

	success := LoginToGpio(&gpioCfg)

	assert.EqualValues(t, true, success)
}

func Test_NewGpioPollService(t *testing.T) {
	teardown := setupGpioTest(false, true, "")
	defer teardown()

	gpioService = NewGpioPollService(&gpioCfg)

	assert.EqualValues(t, gpioCfg, *gpioService.Cfg)
}

func Test_PollRun_NoData_ReturnsError(t *testing.T) {
	teardown := setupGpioTest(false, true, "")
	defer teardown()
	service := NewGpioPollService(&gpioCfg)

	err := service.PollRun()

	assert.NotNil(t, err)
	assert.EqualValues(t, "EOF", err.Error())
}

func Test_stringToBool_ReturnsFalse(t *testing.T) {
	res := stringToBool("0")

	assert.EqualValues(t, false, res)
}

func Test_stringToBool_ReturnsTrue(t *testing.T) {
	res := stringToBool("1")

	assert.EqualValues(t, true, res)
}

func Test_boolToInt_ReturnsOne(t *testing.T) {
	res := boolToInt(true)

	assert.EqualValues(t, 1, res)
}

func Test_boolToInt_ReturnsZero(t *testing.T) {
	res := boolToInt(false)

	assert.EqualValues(t, 0, res)
}

func Test_GetXmlFromPollUrl_GetReqFails_ReturnsError(t *testing.T) {
	teardown := setupGpioTest(true, false, "")
	defer teardown()

	cookie = &http.Cookie{}
	data, err := GetXmlFromPollUrl(server.URL)

	assert.Nil(t, data)
	assert.NotNil(t, err)
	assert.EqualValues(t, "URl not found", err.Error())
}

func Test_GetXmlFromPollUrl_WrongData_ReturnsError(t *testing.T) {
	teardown := setupGpioTest(false, false, "abcdefg")
	defer teardown()

	cookie = &http.Cookie{}
	data, err := GetXmlFromPollUrl(server.URL)

	assert.Nil(t, data)
	assert.NotNil(t, err)
	assert.EqualValues(t, "EOF", err.Error())
}

func Test_GetXmlFromPollUrl_NoError_ReturnsData(t *testing.T) {
	xmlData, _ := os.ReadFile("gpioStat_sample.txt")

	teardown := setupGpioTest(false, false, string(xmlData))
	defer teardown()

	cookie = &http.Cookie{}
	data, err := GetXmlFromPollUrl(server.URL)

	assert.NotNil(t, data)
	assert.Nil(t, err)
	assert.EqualValues(t, "1", data.In[0])
	assert.EqualValues(t, "0", data.In[22])
}

func Test_mapState_MapsCorrectly(t *testing.T) {
	var gpioState domain.DevStat
	gpioState.In = append(gpioState.In, "1")
	gpioState.In = append(gpioState.In, "0")
	gpioState.In = append(gpioState.In, "1")
	gpio1 := config.PinData{
		Id:     1,
		Name:   "Test 1",
		Invert: false,
		State:  false,
	}
	gpio2 := config.PinData{
		Id:     2,
		Name:   "Test 2",
		Invert: false,
		State:  false,
	}
	gpio3 := config.PinData{
		Id:     3,
		Name:   "Test 3",
		Invert: true,
		State:  false,
	}

	gpioCfg.RunTime.Gpios = append(gpioCfg.RunTime.Gpios, gpio1)
	gpioCfg.RunTime.Gpios = append(gpioCfg.RunTime.Gpios, gpio2)
	gpioCfg.RunTime.Gpios = append(gpioCfg.RunTime.Gpios, gpio3)

	mapState(&gpioState, &gpioCfg)

	assert.EqualValues(t, true, gpioCfg.RunTime.Gpios[0].State)
	assert.EqualValues(t, false, gpioCfg.RunTime.Gpios[1].State)
	assert.EqualValues(t, false, gpioCfg.RunTime.Gpios[2].State)
}
