package config

import (
	"bufio"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testEnvFile string = ".testenv"
	testConfig  AppConfig
)

func checkErr(err error) {
	if err != nil {
		panic(fmt.Sprintf("could not execute test preparation.. Error: %s", err))
	}
}

func writeTestEnv(fileName string) {
	f, err := os.Create(fileName)
	checkErr(err)
	defer f.Close()
	w := bufio.NewWriter(f)
	_, err = w.WriteString("GIN_MODE=\"debug\"\n")
	checkErr(err)
	_, err = w.WriteString("SERVER_HOST=\"127.0.0.1\"\n")
	checkErr(err)
	_, err = w.WriteString("SERVER_PORT=\"9999\"\n")
	checkErr(err)
	w.Flush()
}

func deleteEnvFile(fileName string) {
	err := os.Remove(fileName)
	checkErr(err)
}

func unsetEnvVars() {
	os.Unsetenv("GIN_MODE")
	os.Unsetenv("SERVER_HOST")
	os.Unsetenv("SERVER_PORT")
}

func Test_loadConfig_NoEnvFile_Returns_Error(t *testing.T) {
	err := loadConfig("file_does_not_exist.txt")
	assert.NotNil(t, err)
	fmt.Printf("error: %v", err)

	assert.EqualValues(t, "open file_does_not_exist.txt: The system cannot find the file specified.", err.Error())
}

func Test_loadConfig_WithEnvFile_Returns_NoError(t *testing.T) {
	writeTestEnv(testEnvFile)
	defer deleteEnvFile(testEnvFile)
	err := loadConfig(testEnvFile)
	defer unsetEnvVars()

	assert.Nil(t, err)
	assert.EqualValues(t, "127.0.0.1", os.Getenv("SERVER_HOST"))
	assert.EqualValues(t, "debug", os.Getenv("GIN_MODE"))
}

func Test_InitConfig_WithEnvFile_SetsValues(t *testing.T) {
	writeTestEnv(testEnvFile)
	defer deleteEnvFile(testEnvFile)
	err := InitConfig(testEnvFile, &testConfig)

	assert.Nil(t, err)
	assert.EqualValues(t, 10, testConfig.Server.GracefulShutdownTime)
	assert.EqualValues(t, "debug", testConfig.Gin.Mode)
}

func Test_Decode_InvalidString_ReturnsError(t *testing.T) {
	var dec PinConfigDecoder
	var teststring = "this will not work"
	err := dec.Decode(teststring)

	assert.NotNil(t, err)
	assert.EqualValues(t, "invalid map item: \"this will not work\"", err.Error())
}

func Test_Decode_InvalidIndex_ReturnsError(t *testing.T) {
	var dec PinConfigDecoder
	var teststring = "A={\"name\":\"SD1 Master Alarm\",\"invert\": true}"
	err := dec.Decode(teststring)

	assert.NotNil(t, err)
	assert.EqualValues(t, "invalid map index: \"A\"", err.Error())
}

func Test_Decode_InvalidJson_ReturnsError(t *testing.T) {
	var dec PinConfigDecoder
	var teststring = "1={no json to be found}"
	err := dec.Decode(teststring)

	assert.NotNil(t, err)
	assert.EqualValues(t, "invalid map json: invalid character 'n' looking for beginning of object key string", err.Error())
}

func Test_Decode_ValidaData_ReturnsNoError(t *testing.T) {
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

func Test_setupGpios_EmptyConfig_ReturnsEmptyGpioData(t *testing.T) {
	var emptyCfg AppConfig
	SetupGpios(&emptyCfg)

	assert.EqualValues(t, 0, len(emptyCfg.RunTime.Gpios))
}

func Test_setupGpios_2Gpios_ReturnsGpioData(t *testing.T) {
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
