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

type DefaultStreamScrapeService struct {
	Cfg *config.AppConfig
}

var (
	httpTr     http.Transport
	httpClient http.Client
)

func NewScrapeService(cfg *config.AppConfig) DefaultStreamScrapeService {
	InitHttp()
	return DefaultStreamScrapeService{
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

func (s DefaultStreamScrapeService) Scrape() {
	logger.Info(fmt.Sprintf("Starting to scrape %v", s.Cfg.StreamScrape.Url))

	for {
		ScrapeRun(s)
		time.Sleep(time.Duration(s.Cfg.StreamScrape.IntervalSec) * time.Second)
	}
}

func ScrapeRun(s DefaultStreamScrapeService) {
	var streamData domain.IceCastStats
	s.Cfg.RunTime.StreamScrapeCount++
	s.Cfg.Metrics.StreamScrapeCount.Inc()
	body, err := GetDataFromUrl(s.Cfg.StreamScrape.Url)
	if err == nil {
		sanitizedBody := sanitize(body)
		streamData, err = unMarshall(sanitizedBody)
		if err == nil {
			streamCount := UpdateMetrics(streamData, s)
			if streamCount != s.Cfg.StreamScrape.NumExpected {
				logger.Warn(fmt.Sprintf("Expected %v streams, but received %v", s.Cfg.StreamScrape.NumExpected, streamCount))
			}
		}
	}
}

func GetDataFromUrl(Url string) ([]byte, error) {
	resp, err := httpClient.Get(Url)
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

func UpdateMetrics(streamData domain.IceCastStats, s DefaultStreamScrapeService) int {
	streamCount := 0
	for _, source := range streamData.Icestats.Source {
		if source.ServerName == s.Cfg.StreamScrape.ExpectedServerName {
			streamCount++
			name := domain.StreamNames[source.Listenurl]
			listeners := source.Listeners
			s.Cfg.Metrics.StreamListenerGauge.WithLabelValues(name).Set(float64(listeners))
		}
	}
	return streamCount
}
