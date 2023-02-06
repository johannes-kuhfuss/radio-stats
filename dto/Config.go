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
	ScrapeUrl                  string
	ScrapeIntervalSec          string
	ScrapeCount                string
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
		ScrapeUrl:                  cfg.Scrape.Url,
		ScrapeIntervalSec:          strconv.Itoa(cfg.Scrape.IntervalSec),
		ScrapeCount:                strconv.FormatUint(cfg.RunTime.ScrapeCount, 10),
	}
	if cfg.Server.Host == "" {
		resp.ServerHost = "localhost"
	}
	return resp
}
