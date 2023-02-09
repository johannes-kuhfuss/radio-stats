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

	for {
		s.Cfg.RunTime.ScrapeCount++
		s.Cfg.Metrics.ScrapeCount.Inc()
		body, err := GetDataFromUrl(s)
		if err == nil {
			sanitizedBody := sanitize(body)
			streamData, err = unMarshall(sanitizedBody)
			if err == nil {
				streamCount := UpdateMetrics(streamData, s)
				if streamCount != s.Cfg.Scrape.NumExpected {
					logger.Warn(fmt.Sprintf("Expected %v streams, but received %v", s.Cfg.Scrape.NumExpected, streamCount))
				}
			}
		}
		time.Sleep(time.Duration(s.Cfg.Scrape.IntervalSec) * time.Second)
	}
}

func GetDataFromUrl(s DefaultScrapeService) ([]byte, error) {
	resp, err := httpClient.Get(s.Cfg.Scrape.Url)
	if err != nil {
		logger.Error("Error while scraping", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error while reading scrape response", err)
		return nil, err
	}
	return body, nil
}

func sanitize(body []byte) string {
	saniBody := strings.ReplaceAll(string(body), " - ", "null")
	return saniBody
}

func unMarshall(sanitizedBody string) (domain.IceCastStats, error) {
	var streamData domain.IceCastStats
	err := json.Unmarshal([]byte(sanitizedBody), &streamData)
	if err != nil {
		logger.Error("Error while converting to JSON", err)
		return streamData, err
	}
	return streamData, nil
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
