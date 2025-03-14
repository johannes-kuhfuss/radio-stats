// package dto defines the data structures used to exchange information
package dto

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/johannes-kuhfuss/radio-stats/config"
)

// ConfigResp holds converted configuration data for display on the web UI
type ConfigResp struct {
	ServerHost                 string
	ServerPort                 string
	ServerTlsPort              string
	ServerGracefulShutdownTime string
	ServerUseTls               string
	ServerCertFile             string
	ServerKeyFile              string
	GinMode                    string
	StartDate                  string
	StreamScrapeUrl            string
	StreamScrapeIntervalSec    string
	StreamScrapeCount          string
	GpioHost                   string
	GpioConnected              string
	GpioPollIntervalSec        string
	GpioPins                   []struct {
		Id     string
		Name   string
		Invert string
		State  string
	}
	KsPins []struct {
		Id     string
		Name   string
		Invert string
		State  string
	}
	GpioOuts                      []string
	StreamVolDetectionIntervalSec string
	StreamVolDetectionDuration    string
	StreamVolDetectionCount       string
	StreamVolumes                 string
}

// stateBoolToString converts boolean state information to a display string
func stateBoolToString(state bool) string {
	if state {
		return "Active"
	}
	return "Inactive"
}

// invertBoolToString converts boolean inversion information to a display string
func invertBoolToString(state bool) string {
	if state {
		return "Inverted"
	}
	return "Non inverted"
}

// volumeString converts volume information to a display string
func volumeString(volumes map[string]float64) string {
	b := new(bytes.Buffer)
	sUrls := make([]string, 0, len(volumes))
	for k := range volumes {
		sUrls = append(sUrls, k)
	}
	sort.Strings(sUrls)
	for _, sUrl := range sUrls {
		fmt.Fprintf(b, "%s=%s # ", sUrl, strconv.FormatFloat(volumes[sUrl], 'f', -1, 64))
	}
	return b.String()
}

// GetConfig converts the configuration to its display format
func GetConfig(cfg *config.AppConfig) (resp ConfigResp) {
	resp = ConfigResp{
		ServerHost:                    cfg.Server.Host,
		ServerPort:                    cfg.Server.Port,
		ServerTlsPort:                 cfg.Server.TlsPort,
		ServerGracefulShutdownTime:    strconv.Itoa(cfg.Server.GracefulShutdownTime),
		ServerUseTls:                  strconv.FormatBool(cfg.Server.UseTls),
		ServerCertFile:                cfg.Server.CertFile,
		ServerKeyFile:                 cfg.Server.KeyFile,
		GinMode:                       cfg.Gin.Mode,
		StartDate:                     cfg.RunTime.StartDate.Local().Format("2006-01-02 15:04:05 -0700"),
		StreamScrapeUrl:               cfg.StreamScrape.Url,
		StreamScrapeIntervalSec:       strconv.Itoa(cfg.StreamScrape.IntervalSec),
		StreamScrapeCount:             strconv.FormatUint(cfg.RunTime.StreamScrapeCount, 10),
		GpioHost:                      cfg.Gpio.Host,
		GpioConnected:                 strconv.FormatBool(cfg.RunTime.GpioConnected),
		GpioPollIntervalSec:           strconv.Itoa(cfg.Gpio.IntervalSec),
		StreamVolDetectionIntervalSec: strconv.Itoa(cfg.StreamVolDetect.IntervalSec),
		StreamVolDetectionDuration:    strconv.Itoa(cfg.StreamVolDetect.Duration),
		StreamVolDetectionCount:       strconv.FormatUint(cfg.RunTime.StreamVolDetectCount, 10),
	}
	if cfg.Server.Host == "" {
		resp.ServerHost = "localhost"
	}
	cfg.RunTime.StreamVolumes.Lock()
	resp.StreamVolumes = volumeString(cfg.RunTime.StreamVolumes.Vols)
	cfg.RunTime.StreamVolumes.Unlock()
	for _, v := range cfg.RunTime.Gpios {
		var pinData struct {
			Id     string
			Name   string
			Invert string
			State  string
		}
		pinData.Id = strconv.Itoa(v.Id)
		pinData.Name = v.Name
		pinData.Invert = invertBoolToString(v.Invert)
		pinData.State = stateBoolToString(v.State)
		resp.GpioPins = append(resp.GpioPins, pinData)
		if strings.Contains(pinData.Name, "KS") {
			resp.KsPins = append(resp.KsPins, pinData)
		}
	}
	for s := range cfg.Gpio.OutConfig {
		resp.GpioOuts = append(resp.GpioOuts, s)
		sort.Strings(resp.GpioOuts)
	}
	return
}
