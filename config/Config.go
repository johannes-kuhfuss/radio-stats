package config

import (
	"fmt"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/services_utils/api_error"
	"github.com/johannes-kuhfuss/services_utils/logger"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
)

type PinData struct {
	Id    int
	Name  string
	State bool
}

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
	StreamVolDetect struct {
		Url         string `envconfig:"STREAM_VOLDETECT_URL"`
		IntervalSec int    `envconfig:"STREAM_VOLDETECT_INTERVAL_SEC" default:"5"`
		Duration    int    `envconfig:"STREAM_VOLDETECT_DURATION" default:"2"`
		FfmpegExe   string `envconfig:"STREAM_VOLDETECT_FFMPEG" default:"./prog/ffmpeg.exe"`
	}
	Gpio struct {
		Host        string         `envconfig:"GPIO_HOST"`
		User        string         `envconfig:"GPIO_USER" default:"reader"`
		Password    string         `envconfig:"GPIO_PASSWORD" default:"reader"`
		IntervalSec int            `envconfig:"GPIO_POLL_INTERVAL_SEC" default:"1"`
		Names       map[int]string `envconfig:"GPIO_NAMES"`
	}
	Metrics struct {
		StreamListenerGauge  prometheus.GaugeVec
		StreamScrapeCount    prometheus.Counter
		GpioStateGauge       prometheus.GaugeVec
		StreamVolDetectCount prometheus.Counter
		StreamVolume         prometheus.GaugeVec
	}
	RunTime struct {
		Router               *gin.Engine
		ListenAddr           string
		StartDate            time.Time
		StreamScrapeCount    uint64
		StreamVolDetectCount uint64
		StreamVolume         float64
		RunScrape            bool
		RunPoll              bool
		RunListen            bool
		Gpios                []PinData
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
	for key, val := range config.Gpio.Names {
		var gpio PinData
		gpio.Id = key
		gpio.Name = val
		gpio.State = false
		config.RunTime.Gpios = append(config.RunTime.Gpios, gpio)
	}
	sort.SliceStable(config.RunTime.Gpios, func(i, j int) bool {
		return config.RunTime.Gpios[i].Id < config.RunTime.Gpios[j].Id
	})

	config.RunTime.StreamScrapeCount = 0
	config.RunTime.RunScrape = false
	config.RunTime.RunListen = false
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
