// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"context"
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

type StreamScraper interface {
	Scrape()
	ScrapeContext(context.Context)
}

type DefaultStreamScrapeService struct {
	Cfg    *config.AppConfig
	Client *http.Client
}

func NewStreamScrapeService(cfg *config.AppConfig) DefaultStreamScrapeService {
	return DefaultStreamScrapeService{
		Cfg:    cfg,
		Client: NewStreamHttpClient(),
	}
}

func NewStreamHttpClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:  false,
			DisableCompression: false,
			MaxIdleConns:       0,
			IdleConnTimeout:    0,
		},
		Timeout: 5 * time.Second,
	}
}

func InitStreamHttp() {
}

func (s DefaultStreamScrapeService) Scrape() {
	s.ScrapeContext(context.Background())
}

func (s DefaultStreamScrapeService) ScrapeContext(ctx context.Context) {
	if s.Client == nil {
		s.Client = NewStreamHttpClient()
	}
	if s.Cfg.StreamScrape.Url == "" {
		logger.Warn("No scrape URL given. Not scraping stream metrics")
		s.Cfg.SetRunScrape(false)
	} else {
		logger.Info(fmt.Sprintf("Starting to scrape stream metrics from %v", s.Cfg.StreamScrape.Url))
		s.Cfg.SetRunScrape(true)
	}

	ticker := time.NewTicker(intervalSeconds(s.Cfg.StreamScrape.IntervalSec))
	defer ticker.Stop()
	for s.Cfg.ShouldRunScrape() {
		select {
		case <-ctx.Done():
			s.Cfg.SetRunScrape(false)
			return
		default:
		}
		s.ScrapeRun()
		select {
		case <-ctx.Done():
			s.Cfg.SetRunScrape(false)
			return
		case <-ticker.C:
		}
	}
}

func (s DefaultStreamScrapeService) ScrapeRun() {
	var streamData domain.IceCastStats
	s.Cfg.IncStreamScrapeCount()
	s.Cfg.Metrics.StreamScrapeCount.Inc()
	body, err := s.GetDataFromStreamUrl(s.Cfg.StreamScrape.Url)
	if err == nil {
		sanitizedBody := sanitize(body)
		streamData, err = unMarshall(sanitizedBody)
		if err == nil {
			streamCount := s.updateStreamMetrics(streamData)
			if streamCount != s.Cfg.StreamScrape.NumExpected {
				logger.Warn(fmt.Sprintf("Expected %v streams, but received %v", s.Cfg.StreamScrape.NumExpected, streamCount))
			}
		}
	}
}

func GetDataFromStreamUrl(Url string) ([]byte, error) {
	return DefaultStreamScrapeService{Client: NewStreamHttpClient()}.GetDataFromStreamUrl(Url)
}

func (s DefaultStreamScrapeService) GetDataFromStreamUrl(Url string) ([]byte, error) {
	if s.Client == nil {
		s.Client = NewStreamHttpClient()
	}
	resp, err := s.Client.Get(Url)
	if err != nil {
		logger.Error("Error while scraping stream metrics", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		err := fmt.Errorf("stream scrape host returned status %v", resp.Status)
		logger.Error("Error while scraping stream metrics", err)
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error while reading scrape response for stream metrics", err)
		return nil, err
	}
	return body, nil
}

func sanitize(body []byte) string {
	return strings.ReplaceAll(string(body), " - ", "null")
}

func unMarshall(sanitizedBody string) (streamData domain.IceCastStats, e error) {
	if err := json.Unmarshal([]byte(sanitizedBody), &streamData); err != nil {
		logger.Error("Error while converting stream metrics to JSON", err)
		return streamData, err
	}
	return streamData, nil
}

func (s DefaultStreamScrapeService) updateStreamMetrics(streamData domain.IceCastStats) (streamCount int) {
	for _, source := range streamData.Icestats.Source {
		if source.ServerName == s.Cfg.StreamScrape.ExpectedServerName {
			streamCount++
			name := domain.StreamNames[source.Listenurl]
			listeners := source.Listeners
			s.Cfg.Metrics.StreamListenerGauge.WithLabelValues(name).Set(float64(listeners))
		}
	}
	return
}
