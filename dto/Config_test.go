package dto

import (
	"testing"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/stretchr/testify/assert"
)

var (
	testConfig config.AppConfig
)

func Test_GetConfig_Returns_NoError(t *testing.T) {
	config.InitConfig("", &testConfig)
	resp := GetConfig(&testConfig)

	assert.NotNil(t, resp)

	assert.EqualValues(t, "release", resp.GinMode)
	assert.EqualValues(t, "localhost", resp.ServerHost)
}

func Test_boolToStringState_True(t *testing.T) {
	res := stateBoolToString(true)
	assert.EqualValues(t, "Active", res)

}

func Test_boolToStringState_False(t *testing.T) {
	res := stateBoolToString(false)
	assert.EqualValues(t, "Inactive", res)

}
