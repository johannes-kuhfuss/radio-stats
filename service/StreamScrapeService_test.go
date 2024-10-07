package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/radio-stats/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

var (
	scrapeRecorder     *httptest.ResponseRecorder
	scrapeResponseBody string
	scrapeService      DefaultStreamScrapeService
)

func setupScrapeTest() func() {
	scrapeRecorder = httptest.NewRecorder()
	fc, _ := os.ReadFile("../samples/icecast_response.txt")
	scrapeResponseBody = string(fc)
	return func() {
	}
}

func Test_GetDatafromUrl_NoUrl_ReturnsError(t *testing.T) {
	teardown := setupScrapeTest()
	defer teardown()

	body, err := GetDataFromStreamUrl("")

	assert.Nil(t, body)
	assert.NotNil(t, err)
	assert.EqualValues(t, "Get \"\": unsupported protocol scheme \"\"", err.Error())
}

func Test_GetDatafromUrl_ReturnsNoError(t *testing.T) {
	teardown := setupScrapeTest()
	defer teardown()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, scrapeResponseBody)
	}))
	defer server.Close()

	body, err := GetDataFromStreamUrl(server.URL)

	assert.NotNil(t, body)
	assert.Nil(t, err)
}

func Test_sanitize(t *testing.T) {
	testString := "abc - def"
	saniString := sanitize([]byte(testString))
	assert.NotNil(t, saniString)
	assert.EqualValues(t, "abcnulldef", saniString)
}

func Test_unMarshall_Returns_Error(t *testing.T) {
	body := "nojson"
	_, err := unMarshall(body)

	assert.NotNil(t, err)
	assert.EqualValues(t, "invalid character 'o' in literal null (expecting 'u')", err.Error())
}

func Test_unMarshall_Returns_NoError(t *testing.T) {
	stats := domain.IceCastStats{
		Icestats: struct {
			Admin              string "json:\"admin\""
			Host               string "json:\"host\""
			Location           string "json:\"location\""
			ServerID           string "json:\"server_id\""
			ServerStart        string "json:\"server_start\""
			ServerStartIso8601 string "json:\"server_start_iso8601\""
			Source             []struct {
				AudioInfo          string      "json:\"audio_info,omitempty\""
				Channels           int         "json:\"channels,omitempty\""
				Genre              string      "json:\"genre,omitempty\""
				ListenerPeak       int         "json:\"listener_peak,omitempty\""
				Listeners          int         "json:\"listeners\""
				Listenurl          string      "json:\"listenurl\""
				Samplerate         int         "json:\"samplerate,omitempty\""
				ServerDescription  string      "json:\"server_description,omitempty\""
				ServerName         string      "json:\"server_name,omitempty\""
				ServerType         string      "json:\"server_type,omitempty\""
				ServerURL          string      "json:\"server_url,omitempty\""
				StreamStart        string      "json:\"stream_start,omitempty\""
				StreamStartIso8601 string      "json:\"stream_start_iso8601,omitempty\""
				Title              string      "json:\"title,omitempty\""
				Dummy              interface{} "json:\"dummy\""
				Artist             interface{} "json:\"artist,omitempty\""
				AudioBitrate       int         "json:\"audio_bitrate,omitempty\""
				AudioChannels      int         "json:\"audio_channels,omitempty\""
				AudioSamplerate    int         "json:\"audio_samplerate,omitempty\""
				Bitrate            interface{} "json:\"bitrate,omitempty\""
				IceBitrate         int         "json:\"ice-bitrate,omitempty\""
				IceChannels        int         "json:\"ice-channels,omitempty\""
				IceSamplerate      int         "json:\"ice-samplerate,omitempty\""
				Subtype            string      "json:\"subtype,omitempty\""
				Quality            float64     "json:\"quality,omitempty\""
			} "json:\"source\""
		}{},
	}
	statsJson, _ := json.Marshal(stats)
	body, err := unMarshall(string(statsJson))

	assert.Nil(t, err)
	assert.NotNil(t, body)
	assert.EqualValues(t, stats, body)
}

func Test_Scrape_NoUrl_DoesntScrape(t *testing.T) {
	var cfg config.AppConfig
	scrapeService = NewStreamScrapeService(&cfg)
	scrapeService.Scrape()

	assert.EqualValues(t, false, cfg.RunTime.RunScrape)
}

func Test_ScrapeRun_LocalUrl_UpdatesMetrics(t *testing.T) {
	var cfg config.AppConfig

	teardown := setupScrapeTest()
	defer teardown()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, scrapeResponseBody)
	}))
	defer server.Close()

	cfg.StreamScrape.Url = server.URL
	cfg.Metrics.StreamScrapeCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "scrape_count",
		Help:      "Number of times stream count data was retrieved from streaming server",
	})
	cfg.Metrics.StreamListenerGauge = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "Streams",
		Name:      "listener_count",
		Help:      "Number of listeners per stream",
	}, []string{
		"streamName",
	})
	prometheus.MustRegister(cfg.Metrics.StreamListenerGauge)
	prometheus.MustRegister(cfg.Metrics.StreamScrapeCount)

	scrapeService = NewStreamScrapeService(&cfg)
	scrapeService.ScrapeRun()

	assert.EqualValues(t, 1, cfg.RunTime.StreamScrapeCount)
}
