package config

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/services_utils/api_error"
	"github.com/johannes-kuhfuss/services_utils/logger"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
)

type AppConfig struct {
	Server struct {
		Host                 string `envconfig:"SERVER_HOST"`
		Port                 string `envconfig:"SERVER_PORT" default:"8080"`
		TlsPort              string `envconfig:"SERVER_TLS_PORT" default:"8443"`
		GracefulShutdownTime int    `envconfig:"GRACEFUL_SHUTDOWN_TIME" default:"10"`
		UseTls               bool   `envconfig:"USE_TLS" default:"false"`
		CertFile             string `envconfig:"CERT_FILE" default:"./cert/cert.pem"`
		KeyFile              string `envconfig:"KEY_FILE" default:"./cert/cert.key"`
	}
	Gin struct {
		Mode         string `envconfig:"GIN_MODE" default:"release"`
		TemplatePath string `envconfig:"TEMPLATE_PATH" default:"./templates/"`
	}
	StreamScrape struct {
		Url                string `envconfig:"STREAM_SCRAPE_URL"`
		IntervalSec        int    `envconfig:"STREAM_SCRAPE_INTERVAL_SEC" default:"5"`
		NumExpected        int    `envconfig:"NUM_STREAMS_EXPECTED" default:"5"`
		ExpectedServerName string `envconfig:"EXPECTED_SERVER_NAME" default:"coloRadio"`
	}
	Gpio struct {
		SerialPort          string `envconfig:"GPIO_SERIAL_PORT"`
		GpioPollIntervalSec int    `envconfig:"GPIO_POLL_INTERVAL_SEC" default:"1"`
		Gpio01Name          string `envconfig:"GPIO_01_NAME" default:"IO 01"`
		Gpio02Name          string `envconfig:"GPIO_02_NAME" default:"IO 02"`
		Gpio03Name          string `envconfig:"GPIO_03_NAME" default:"IO 03"`
		Gpio04Name          string `envconfig:"GPIO_04_NAME" default:"IO 04"`
		Gpio05Name          string `envconfig:"GPIO_05_NAME" default:"IO 05"`
		Gpio06Name          string `envconfig:"GPIO_06_NAME" default:"IO 06"`
		Gpio07Name          string `envconfig:"GPIO_07_NAME" default:"IO 07"`
		Gpio08Name          string `envconfig:"GPIO_08_NAME" default:"IO 08"`
	}
	Metrics struct {
		StreamListenerGauge prometheus.GaugeVec
		StreamScrapeCount   prometheus.Counter
		GpioStateGauge      prometheus.GaugeVec
	}
	RunTime struct {
		Router            *gin.Engine
		ListenAddr        string
		StartDate         time.Time
		StreamScrapeCount uint64
		Gpio01State       bool
		Gpio02State       bool
		Gpio03State       bool
		Gpio04State       bool
		Gpio05State       bool
		Gpio06State       bool
		Gpio07State       bool
		Gpio08State       bool
	}
}

var (
	EnvFile = ".env"
)

func InitConfig(file string, config *AppConfig) api_error.ApiErr {
	logger.Info(fmt.Sprintf("Initalizing configuration from file %v", file))
	loadConfig(file)
	err := envconfig.Process("", config)
	if err != nil {
		return api_error.NewInternalServerError("Could not initalize configuration. Check your environment variables", err)
	}
	config.RunTime.StreamScrapeCount = 0
	config.RunTime.Gpio01State = false
	config.RunTime.Gpio02State = false
	config.RunTime.Gpio03State = false
	config.RunTime.Gpio04State = false
	config.RunTime.Gpio05State = false
	config.RunTime.Gpio06State = false
	config.RunTime.Gpio07State = false
	config.RunTime.Gpio08State = false
	logger.Info("Done initalizing configuration")
	return nil
}

func loadConfig(file string) error {
	err := godotenv.Load(file)
	if err != nil {
		logger.Info("Could not open env file. Using Environment variable and defaults")
		return err
	}
	return nil
}
