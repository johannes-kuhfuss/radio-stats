package config

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/emberplus/client"
	"github.com/johannes-kuhfuss/services_utils/api_error"
	"github.com/johannes-kuhfuss/services_utils/logger"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
)

type PinConfig struct {
	Name   string
	Invert bool
}

type PinData struct {
	Id     int
	Name   string
	Invert bool
	State  bool
}

type PinConfigDecoder map[int]PinConfig

func (pdd *PinConfigDecoder) Decode(value string) error {
	gpioData := map[int]PinConfig{}
	pins := strings.Split(value, ";")
	for _, pin := range pins {
		pinData := PinConfig{}
		kvpair := strings.Split(pin, "=")
		if len(kvpair) != 2 {
			return fmt.Errorf("invalid map item: %q", pin)
		}
		index, err := strconv.Atoi(kvpair[0])
		if err != nil {
			return fmt.Errorf("invalid map index: %q", kvpair[0])
		}
		err = json.Unmarshal([]byte(kvpair[1]), &pinData)
		if err != nil {
			return fmt.Errorf("invalid map json: %w", err)
		}
		gpioData[index] = pinData
	}
	*pdd = PinConfigDecoder(gpioData)
	return nil
}

type EmberConfig struct {
	Port          int
	EntryPath     string
	MetricsPrefix string
	GPIOs         []string
	Conn          *client.EmberClient
}

type EmberConfigDecoder map[string]EmberConfig

func (ed *EmberConfigDecoder) Decode(value string) error {
	emberData := map[string]EmberConfig{}
	hosts := strings.Split(value, ";")
	for _, host := range hosts {
		hostData := EmberConfig{}
		kvpair := strings.Split(host, "=")
		if len(kvpair) != 2 {
			return fmt.Errorf("invalid map item: %q", host)
		}
		err := json.Unmarshal([]byte(kvpair[1]), &hostData)
		if err != nil {
			return fmt.Errorf("invalid map json: %w", err)
		}
		emberData[kvpair[0]] = hostData
	}
	*ed = EmberConfigDecoder(emberData)
	return nil
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
		AdminUserName        string `envconfig:"ADMIN_USER_NAME" default:"admin"`
		AdminPasswordHash    string `envconfig:"ADMIN_PASSWORD_HASH"`
	}
	Gin struct {
		Mode         string `envconfig:"GIN_MODE" default:"release"`
		TemplatePath string `envconfig:"TEMPLATE_PATH" default:"./templates/"`
		LogToLogger  bool   `envconfig:"LOG_TO_LOGGER" default:"false"`
	}
	StreamScrape struct {
		Url                string `envconfig:"STREAM_SCRAPE_URL"`
		IntervalSec        int    `envconfig:"STREAM_SCRAPE_INTERVAL_SEC" default:"5"`
		NumExpected        int    `envconfig:"NUM_STREAMS_EXPECTED" default:"5"`
		ExpectedServerName string `envconfig:"EXPECTED_SERVER_NAME" default:"coloRadio"`
	}
	StreamVolDetect struct {
		Urls        []string `envconfig:"STREAM_VOLDETECT_URLS"`
		IntervalSec int      `envconfig:"STREAM_VOLDETECT_INTERVAL_SEC" default:"5"`
		Duration    int      `envconfig:"STREAM_VOLDETECT_DURATION" default:"2"`
		FfmpegExe   string   `envconfig:"STREAM_VOLDETECT_FFMPEG" default:"/usr/bin/ffmpeg"`
	}
	Gpio struct {
		Host        string           `envconfig:"GPIO_HOST"`
		User        string           `envconfig:"GPIO_USER" default:"reader"`
		Password    string           `envconfig:"GPIO_PASSWORD" default:"reader"`
		IntervalSec int              `envconfig:"GPIO_POLL_INTERVAL_SEC" default:"1"`
		InConfig    PinConfigDecoder `envconfig:"GPIO_IN_CONFIG"`
		OutConfig   map[string]int   `envconfig:"GPIO_OUT_CONFIG"`
	}
	Ember struct {
		IntervalSec int                `envconfig:"EMBER_POLL_INTERVAL_SEC" default:"1"`
		InConfig    EmberConfigDecoder `envconfig:"EMBER_IN_CONFIG"`
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
		StreamVolumes        struct {
			sync.Mutex
			Vols map[string]float64
		}
		RunScrape     bool
		RunGpioPoll   bool
		RunEmberPoll  bool
		GpioConnected bool
		RunListen     bool
		Gpios         []PinData
		EmberGpios    map[string]EmberConfig
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
	SetupGpios(config)
	config.RunTime.StreamScrapeCount = 0
	config.RunTime.RunScrape = false
	config.RunTime.RunListen = false
	config.RunTime.StreamVolumes.Vols = make(map[string]float64)
	config.RunTime.EmberGpios = make(map[string]EmberConfig)
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

func SetupGpios(config *AppConfig) {
	for key, val := range config.Gpio.InConfig {
		var gpio PinData
		gpio.Id = key
		gpio.Name = val.Name
		gpio.Invert = val.Invert
		gpio.State = false
		config.RunTime.Gpios = append(config.RunTime.Gpios, gpio)
	}
	sort.SliceStable(config.RunTime.Gpios, func(i, j int) bool {
		return config.RunTime.Gpios[i].Id < config.RunTime.Gpios[j].Id
	})
}
