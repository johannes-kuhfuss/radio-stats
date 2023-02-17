package dto

import (
	"strconv"

	"github.com/johannes-kuhfuss/radio-stats/config"
)

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
	Gpio01State                string
	Gpio02State                string
	Gpio03State                string
	Gpio04State                string
	Gpio05State                string
	Gpio06State                string
	Gpio07State                string
	Gpio08State                string
}

func boolToStringState(state bool) string {
	if state {
		return "Active"
	} else {
		return "Inactive"
	}
}

func GetConfig(cfg *config.AppConfig) ConfigResp {
	resp := ConfigResp{
		ServerHost:                 cfg.Server.Host,
		ServerPort:                 cfg.Server.Port,
		ServerTlsPort:              cfg.Server.TlsPort,
		ServerGracefulShutdownTime: strconv.Itoa(cfg.Server.GracefulShutdownTime),
		ServerUseTls:               strconv.FormatBool(cfg.Server.UseTls),
		ServerCertFile:             cfg.Server.CertFile,
		ServerKeyFile:              cfg.Server.KeyFile,
		GinMode:                    cfg.Gin.Mode,
		StartDate:                  cfg.RunTime.StartDate.Local().Format("2006-01-02 15:04:05 -0700"),
		StreamScrapeUrl:            cfg.StreamScrape.Url,
		StreamScrapeIntervalSec:    strconv.Itoa(cfg.StreamScrape.IntervalSec),
		StreamScrapeCount:          strconv.FormatUint(cfg.RunTime.StreamScrapeCount, 10),
		Gpio01State:                boolToStringState(cfg.RunTime.Gpio01State),
		Gpio02State:                boolToStringState(cfg.RunTime.Gpio02State),
		Gpio03State:                boolToStringState(cfg.RunTime.Gpio03State),
		Gpio04State:                boolToStringState(cfg.RunTime.Gpio04State),
		Gpio05State:                boolToStringState(cfg.RunTime.Gpio05State),
		Gpio06State:                boolToStringState(cfg.RunTime.Gpio06State),
		Gpio07State:                boolToStringState(cfg.RunTime.Gpio07State),
		Gpio08State:                boolToStringState(cfg.RunTime.Gpio08State),
	}
	if cfg.Server.Host == "" {
		resp.ServerHost = "localhost"
	}
	return resp
}
