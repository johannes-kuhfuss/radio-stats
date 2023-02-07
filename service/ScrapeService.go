package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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
	var streamData domain.IceCastStats
	logger.Info(fmt.Sprintf("Starting to scrape %v", s.Cfg.Scrape.Url))

	for !s.Cfg.RunTime.Terminate {
		s.Cfg.RunTime.ScrapeCount++
		s.Cfg.Metrics.ScrapeCount.Inc()
		_, body := GetDataFromUrl(s)
		sanitizedBody := sanitize(body)
		unMarshall(sanitizedBody, streamData)
		streamCount := UpdateMetrics(streamData, s)
		if streamCount != s.Cfg.Scrape.NumExpected {
			logger.Warn(fmt.Sprintf("Expected %v streams, but received %v", s.Cfg.Scrape.NumExpected, streamCount))
		}
		time.Sleep(time.Duration(s.Cfg.Scrape.IntervalSec) * time.Second)
	}
	logger.Info(fmt.Sprintf("Stopping to scrape %v", s.Cfg.Scrape.Url))
}

func UpdateMetrics(streamData domain.IceCastStats, s DefaultScrapeService) int {
	streamCount := 0
	for _, source := range streamData.Icestats.Source {
		if source.ServerName == s.Cfg.Scrape.ExpectedServerName {
			streamCount++
			name := domain.StreamNames[source.Listenurl]
			listeners := source.Listeners
			s.Cfg.Metrics.ListenerGauge.WithLabelValues(name).Set(float64(listeners))
		}
	}
	return streamCount
}

func unMarshall(sanitizedBody string, streamData domain.IceCastStats) {
	err := json.Unmarshal([]byte(sanitizedBody), &streamData)
	if err != nil {
		logger.Error("Error while converting to JSON", err)
	}
}

func sanitize(body []byte) string {
	saniBody := strings.ReplaceAll(string(body), " - ", "null")
	return saniBody
}

func GetDataFromUrl(s DefaultScrapeService) (error, []byte) {
	resp, err := httpClient.Get(s.Cfg.Scrape.Url)
	if err != nil {
		logger.Error("Error while scraping", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error while reading scrape response", err)
	}
	return err, body
}
