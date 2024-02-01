package dto

import (
	"strings"
	"testing"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/stretchr/testify/assert"
)

var (
	testConfig config.AppConfig
)

func Test_stateBoolToString_True(t *testing.T) {
	res := stateBoolToString(true)
	assert.EqualValues(t, "Active", res)

}

func Test_stateBoolToString_False(t *testing.T) {
	res := stateBoolToString(false)
	assert.EqualValues(t, "Inactive", res)
}

func Test_invertBoolToString_True(t *testing.T) {
	res := invertBoolToString(true)
	assert.EqualValues(t, "Inverted", res)

}

func Test_invertBoolToString_False(t *testing.T) {
	res := invertBoolToString(false)
	assert.EqualValues(t, "Non inverted", res)
}

func Test_GetConfig_NoGpios_Returns_NoError(t *testing.T) {
	config.InitConfig("", &testConfig)
	resp := GetConfig(&testConfig)

	assert.NotNil(t, resp)

	assert.EqualValues(t, "release", resp.GinMode)
	assert.EqualValues(t, "localhost", resp.ServerHost)
}

func Test_GetConfig_WithGpios_Returns_NoError(t *testing.T) {
	config.InitConfig("", &testConfig)
	var dec config.PinConfigDecoder
	var teststring = "1={\"name\":\"SD1 Master Alarm\",\"invert\": true};20={\"name\":\"SD1 Aux Alarm\",\"invert\":false};40={\"name\":\"KS9\",\"invert\":false}"
	dec.Decode(teststring)
	testConfig.Gpio.InConfig = dec
	testConfig.Gpio.OutConfig = make(map[string]int)
	testConfig.Gpio.OutConfig["KS1"] = 1
	testConfig.Gpio.OutConfig["KS2"] = 2
	config.SetupGpios(&testConfig)

	resp := GetConfig(&testConfig)

	assert.NotNil(t, resp)

	assert.EqualValues(t, "SD1 Master Alarm", resp.GpioPins[0].Name)
	assert.EqualValues(t, "Inverted", resp.GpioPins[0].Invert)
	assert.EqualValues(t, "Non inverted", resp.GpioPins[1].Invert)
	assert.EqualValues(t, "KS9", resp.KsPins[0].Name)
	assert.EqualValues(t, "KS1 KS2", strings.Join(resp.GpioOuts, " "))
}

func Test_GetConfig_WithGpioOut_returnsNoError(t *testing.T) {

}

func Test_volumeString_ReturnsString(t *testing.T) {
	vol := make(map[string]float64)
	vol["test1"] = 1.1
	vol["test2"] = 2.2

	volStr := volumeString(vol)

	assert.EqualValues(t, "test1=1.1 # test2=2.2 # ", volStr)
}
