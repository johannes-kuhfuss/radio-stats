package config

import (
	"bufio"
	"fmt"
	"net/http"
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
		panic(fmt.Sprintf("could not execute test preparation. Error: %s", err))
	}
}

func writeTestEnv(fileName string) {
	f, err := os.Create(fileName)
	checkErr(err)
	defer f.Close()
	w := bufio.NewWriter(f)
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
}

func Test_InitConfig_NoEnvFile_Returns_Error(t *testing.T) {
	err := InitConfig("file_does_not_exist.txt", &testConfig)

	assert.NotNil(t, err)
	assert.EqualValues(t, http.StatusInternalServerError, err.StatusCode())
	assert.EqualValues(t, "Could not initalize configuration. Check your environment variables", err.Message())
}

func Test_InitConfig_WithEnvFile_SetsValues(t *testing.T) {
	writeTestEnv(testEnvFile)
	defer deleteEnvFile(testEnvFile)
	err := InitConfig(testEnvFile, &testConfig)

	assert.Nil(t, err)
	assert.EqualValues(t, 10, testConfig.Server.GracefulShutdownTime)
}
