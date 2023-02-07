package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/radio-stats/domain"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type ScrapeService interface {
	Scrape()
}

type DefaultScrapeService struct {
	Cfg *config.AppConfig
}

var (
	httpTr     http.Transport
	httpClient http.Client
)

func NewScrapeService(cfg *config.AppConfig) DefaultScrapeService {
	InitHttp()
	return DefaultScrapeService{
		Cfg: cfg,
	}
}

func InitHttp() {
	httpTr = http.Transport{
		DisableKeepAlives:  false,
		DisableCompression: false,
		MaxIdleConns:       0,
		IdleConnTimeout:    0,
	}
	httpClient = http.Client{Transport: &httpTr}
}

func (s DefaultScrapeService) Scrape() {
	var streamData domain.IceStats
	logger.Info(fmt.Sprintf("Starting to scrape %v", s.Cfg.Scrape.Url))
	// get data
	resp, err := httpClient.Get(s.Cfg.Scrape.Url)
	if err != nil {
		logger.Error("Error while scraping", err)
	}
	defer resp.Body.Close()
	// read data from body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error while reading scrape response", err)
	}
	// sanitize data
	saniData := strings.ReplaceAll(string(body), " - ", " \"-\" ")
	// Unmarshal to json
	err = json.Unmarshal([]byte(saniData), &streamData)
	if err != nil {
		logger.Error("Error while converting to JSON", err)
	}
	//logger.Info(string(body))
}
