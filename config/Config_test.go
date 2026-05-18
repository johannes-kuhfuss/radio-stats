package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func writeTestEnv(t *testing.T) string {
	t.Helper()
	fileName := filepath.Join(t.TempDir(), ".testenv")
	err := os.WriteFile(fileName, []byte("GIN_MODE=\"debug\"\nSERVER_HOST=\"127.0.0.1\"\nSERVER_PORT=\"9999\"\n"), 0644)
	assert.Nil(t, err)
	return fileName
}

func TestLoadConfigNoEnvFileReturnsError(t *testing.T) {
	err := loadConfig("file_does_not_exist.txt")
	assert.NotNil(t, err)

	assert.Contains(t, err.Error(), "file_does_not_exist.txt")
}

func TestLoadConfigWithEnvFileReturnsNoError(t *testing.T) {
	os.Unsetenv("GIN_MODE")
	os.Unsetenv("SERVER_HOST")
	os.Unsetenv("SERVER_PORT")
	t.Cleanup(func() {
		os.Unsetenv("GIN_MODE")
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("SERVER_PORT")
	})

	err := loadConfig(writeTestEnv(t))

	assert.Nil(t, err)
	assert.EqualValues(t, "127.0.0.1", os.Getenv("SERVER_HOST"))
	assert.EqualValues(t, "debug", os.Getenv("GIN_MODE"))
}

func TestInitConfigWithEnvFileSetsValues(t *testing.T) {
	os.Unsetenv("GIN_MODE")
	os.Unsetenv("SERVER_HOST")
	os.Unsetenv("SERVER_PORT")
	t.Cleanup(func() {
		os.Unsetenv("GIN_MODE")
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("SERVER_PORT")
	})
	var testConfig AppConfig

	err := InitConfig(writeTestEnv(t), &testConfig)

	assert.Nil(t, err)
	assert.EqualValues(t, 10, testConfig.Server.GracefulShutdownTime)
	assert.EqualValues(t, "debug", testConfig.Gin.Mode)
}

func TestDecodeInvalidStringReturnsError(t *testing.T) {
	var dec PinConfigDecoder
	var teststring = "this will not work"
	err := dec.Decode(teststring)

	assert.NotNil(t, err)
	assert.EqualValues(t, "invalid map item: \"this will not work\"", err.Error())
}

func TestDecodeInvalidIndexReturnsError(t *testing.T) {
	var dec PinConfigDecoder
	var teststring = "A={\"name\":\"SD1 Master Alarm\",\"invert\": true}"
	err := dec.Decode(teststring)

	assert.NotNil(t, err)
	assert.EqualValues(t, "invalid map index: \"A\"", err.Error())
}

func TestDecodeInvalidJsonReturnsError(t *testing.T) {
	var dec PinConfigDecoder
	var teststring = "1={no json to be found}"
	err := dec.Decode(teststring)

	assert.NotNil(t, err)
	assert.EqualValues(t, "invalid map json: invalid character 'n' looking for beginning of object key string", err.Error())
}

func TestDecodeValidaDataReturnsNoError(t *testing.T) {
	var dec PinConfigDecoder
	var teststring = "1={\"name\":\"SD1 Master Alarm\",\"invert\": true};20={\"name\":\"SD1 Aux Alarm\",\"invert\":false}"
	err := dec.Decode(teststring)

	assert.Nil(t, err)
	assert.EqualValues(t, 2, len(dec))
	assert.EqualValues(t, "SD1 Master Alarm", dec[1].Name)
	assert.EqualValues(t, "SD1 Aux Alarm", dec[20].Name)
	assert.EqualValues(t, true, dec[1].Invert)
	assert.EqualValues(t, false, dec[20].Invert)
}

func TestEmberDecodeInvalidStringReturnsError(t *testing.T) {
	var dec EmberConfigDecoder

	err := dec.Decode("this will not work")

	assert.NotNil(t, err)
	assert.EqualValues(t, "invalid map item: \"this will not work\"", err.Error())
}

func TestEmberDecodeInvalidJsonReturnsError(t *testing.T) {
	var dec EmberConfigDecoder

	err := dec.Decode("host={no json to be found}")

	assert.NotNil(t, err)
	assert.EqualValues(t, "invalid map json: invalid character 'n' looking for beginning of object key string", err.Error())
}

func TestEmberDecodeValidDataReturnsNoError(t *testing.T) {
	var dec EmberConfigDecoder

	err := dec.Decode("host={\"port\":9000,\"entrypath\":\"1.2.3\",\"metricsprefix\":\"ember_\",\"gpios\":[\"1\",\"2\"]}")

	assert.Nil(t, err)
	assert.EqualValues(t, 1, len(dec))
	assert.EqualValues(t, 9000, dec["host"].Port)
	assert.EqualValues(t, "1.2.3", dec["host"].EntryPath)
	assert.EqualValues(t, "ember_", dec["host"].MetricsPrefix)
	assert.EqualValues(t, []string{"1", "2"}, dec["host"].GPIOs)
}

func TestSetupGpiosEmptyConfigReturnsEmptyGpioData(t *testing.T) {
	var emptyCfg AppConfig
	SetupGpios(&emptyCfg)

	assert.EqualValues(t, 0, len(emptyCfg.RunTime.Gpios))
}

func TestSetupGpios2GpiosReturnsGpioData(t *testing.T) {
	var cfg AppConfig
	var dec PinConfigDecoder
	var teststring = "1={\"name\":\"SD1 Master Alarm\",\"invert\": true};20={\"name\":\"SD1 Aux Alarm\",\"invert\":false}"
	dec.Decode(teststring)
	cfg.Gpio.InConfig = dec

	SetupGpios(&cfg)

	assert.EqualValues(t, 2, len(cfg.RunTime.Gpios))
	assert.EqualValues(t, "SD1 Master Alarm", cfg.RunTime.Gpios[0].Name)
	assert.EqualValues(t, 20, cfg.RunTime.Gpios[1].Id)
	assert.EqualValues(t, false, cfg.RunTime.Gpios[1].State)
}

func TestRuntimeSettersAndCounters(t *testing.T) {
	var cfg AppConfig

	cfg.SetRunScrape(true)
	cfg.SetRunListen(true)
	cfg.SetRunGpioPoll(true)
	cfg.SetRunEmberPoll(true)
	cfg.SetGpioConnected(true)
	cfg.IncStreamScrapeCount()
	cfg.IncStreamVolDetectCount()

	assert.True(t, cfg.ShouldRunScrape())
	assert.True(t, cfg.ShouldRunListen())
	assert.True(t, cfg.ShouldRunGpioPoll())
	assert.True(t, cfg.ShouldRunEmberPoll())
	assert.True(t, cfg.RunTime.GpioConnected)
	assert.EqualValues(t, 1, cfg.RunTime.StreamScrapeCount)
	assert.EqualValues(t, 1, cfg.RunTime.StreamVolDetectCount)
}
