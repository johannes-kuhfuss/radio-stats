package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-json"
	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/radio-stats/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

var (
	cfg          config.AppConfig
	recorder     *httptest.ResponseRecorder
	responseBody string = "{\"icestats\":{\"admin\":\"icemaster@localhost\",\"host\":\"streaming.fueralle.org\",\"location\":\"Earth\",\"server_id\":\"Icecast 2.4.4\",\"server_start\":\"Mon, 12 Dec 2022 14:46:26 +0100\",\"server_start_iso8601\":\"2022-12-12T14:46:26+0100\",\"source\":[{\"audio_info\":\"channels=2;samplerate=44100;bitrate=160\",\"channels\":2,\"genre\":\"Freies Radio\",\"listener_peak\":32,\"listeners\":28,\"listenurl\":\"http://streaming.fueralle.org:80/Radio-F.R.E.I\",\"samplerate\":44100,\"server_description\":\"Freies Radio Erfurt\",\"server_name\":\"Radio F.R.E.I.\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.radiofrei.de\",\"stream_start\":\"Thu, 09 Feb 2023 09:55:41 +0100\",\"stream_start_iso8601\":\"2023-02-09T09:55:41+0100\",\"dummy\":null},{\"audio_info\":\"channels=1;samplerate=44100;bitrate=56\",\"channels\":1,\"genre\":\"Freies Radio\",\"listener_peak\":2,\"listeners\":0,\"listenurl\":\"http://streaming.fueralle.org:80/Radio-F.R.E.I-Low\",\"samplerate\":44100,\"server_description\":\"Freies Radio Erfurt\",\"server_name\":\"Radio F.R.E.I.\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.radiofrei.de\",\"stream_start\":\"Tue, 07 Feb 2023 02:26:57 +0100\",\"stream_start_iso8601\":\"2023-02-07T02:26:57+0100\",\"title\":\"Radio F.R.E.I.\",\"dummy\":null},{\"artist\":null,\"audio_bitrate\":112000,\"audio_channels\":2,\"audio_info\":\"ice-samplerate=44100;ice-bitrate=Quality 3;ice-channels=2\",\"audio_samplerate\":44100,\"bitrate\":\"Quality 3\",\"genre\":\"Freies Radio, community radio\",\"ice-bitrate\":112,\"ice-channels\":2,\"ice-samplerate\":44100,\"listener_peak\":10,\"listeners\":6,\"listenurl\":\"http://streaming.fueralle.org:80/bermudafunk.ogg\",\"server_description\":\"bermuda.funk - Freies Radio Rhein-Neckar e.V.\",\"server_name\":\"bermuda.funk - Freies Radio Rhein-Neckar e.V.\",\"server_type\":\"application/ogg\",\"server_url\":\"http://www.bermudafunk.org\",\"stream_start\":\"Thu, 09 Feb 2023 04:19:02 +0100\",\"stream_start_iso8601\":\"2023-02-09T04:19:02+0100\",\"subtype\":\"Vorbis\",\"title\":null,\"dummy\":null},{\"audio_info\":\"ice-samplerate=44100;ice-bitrate=128;ice-channels=2\",\"bitrate\":128,\"genre\":\"Freies Radio, community radio\",\"ice-bitrate\":128,\"ice-channels\":2,\"ice-samplerate\":44100,\"listener_peak\":8,\"listeners\":5,\"listenurl\":\"http://streaming.fueralle.org:80/bermudafunk_high\",\"server_description\":\"bermuda.funk - Freies Radio Rhein-Neckar e.V.\",\"server_name\":\"bermuda.funk - Freies Radio Rhein-Neckar e.V.\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.bermudafunk.org\",\"stream_start\":\"Thu, 09 Feb 2023 04:19:10 +0100\",\"stream_start_iso8601\":\"2023-02-09T04:19:10+0100\",\"dummy\":null},{\"audio_info\":\"ice-samplerate=22050;ice-bitrate=56;ice-channels=1\",\"bitrate\":56,\"genre\":\"Freies Radio, community radio\",\"ice-bitrate\":56,\"ice-channels\":1,\"ice-samplerate\":22050,\"listener_peak\":8,\"listeners\":4,\"listenurl\":\"http://streaming.fueralle.org:80/bermudafunk_low\",\"server_description\":\"bermuda.funk - Freies Radio Rhein-Neckar e.V.\",\"server_name\":\"bermuda.funk - Freies Radio Rhein-Neckar e.V.\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.bermudafunk.org\",\"stream_start\":\"Thu, 09 Feb 2023 04:19:09 +0100\",\"stream_start_iso8601\":\"2023-02-09T04:19:09+0100\",\"dummy\":null},{\"audio_info\":\"ice-samplerate=44100;ice-bitrate=160;ice-channels=2\",\"bitrate\":160,\"genre\":\"freies Radio\",\"ice-bitrate\":160,\"ice-channels\":2,\"ice-samplerate\":44100,\"listener_peak\":17,\"listeners\":14,\"listenurl\":\"http://streaming.fueralle.org:80/coloradio_160.mp3\",\"server_description\":\"freie Radios aus Dresden (wochendtags 18-23 / wochende 12-24) und anderswo\",\"server_name\":\"coloRadio\",\"server_type\":\"audio/mpeg\",\"server_url\":\"coloradio.org\",\"stream_start\":\"Thu, 09 Feb 2023 03:35:08 +0100\",\"stream_start_iso8601\":\"2023-02-09T03:35:08+0100\",\"title\":\" - Es lÃ¤uft: Listen2Radio 2018 von JUNGES RADIO\",\"dummy\":null},{\"artist\":null,\"audio_bitrate\":160000,\"audio_channels\":2,\"audio_info\":\"ice-samplerate=44100;ice-bitrate=160;ice-channels=2\",\"audio_samplerate\":44100,\"bitrate\":160,\"genre\":\"freies Radio\",\"ice-bitrate\":160,\"ice-channels\":2,\"ice-samplerate\":44100,\"listener_peak\":1,\"listeners\":0,\"listenurl\":\"http://streaming.fueralle.org:80/coloradio_160.ogg\",\"server_description\":\"freie Radios aus Dresden (wochendtags 18-23 / wochende 12-24) und anderswo\",\"server_name\":\"coloRadio\",\"server_type\":\"application/ogg\",\"server_url\":\"http://www.coloradio.org\",\"stream_start\":\"Thu, 09 Feb 2023 03:35:10 +0100\",\"stream_start_iso8601\":\"2023-02-09T03:35:10+0100\",\"subtype\":\"Vorbis\",\"title\":\"Es läuft: Listen2Radio 2018 von JUNGES RADIO\",\"dummy\":null},{\"audio_info\":\"ice-samplerate=44100;ice-bitrate=48;ice-channels=1\",\"bitrate\":48,\"genre\":\"freies Radio\",\"ice-bitrate\":48,\"ice-channels\":1,\"ice-samplerate\":44100,\"listener_peak\":4,\"listeners\":2,\"listenurl\":\"http://streaming.fueralle.org:80/coloradio_48.aac\",\"server_description\":\"freie Radios aus Dresden (wochendtags 18-23 / wochende 12-24) und anderswo\",\"server_name\":\"coloRadio\",\"server_type\":\"audio/aac\",\"server_url\":\"http://www.coloradio.org\",\"stream_start\":\"Thu, 09 Feb 2023 03:35:11 +0100\",\"stream_start_iso8601\":\"2023-02-09T03:35:11+0100\",\"title\":\" - Es lÃ¤uft: Listen2Radio 2018 von JUNGES RADIO\",\"dummy\":null},{\"audio_info\":\"ice-samplerate=44100;ice-bitrate=56;ice-channels=2\",\"bitrate\":56,\"genre\":\"freies Radio\",\"ice-bitrate\":56,\"ice-channels\":2,\"ice-samplerate\":44100,\"listener_peak\":2,\"listeners\":0,\"listenurl\":\"http://streaming.fueralle.org:80/coloradio_56.mp3\",\"server_description\":\"freie Radios aus Dresden (wochendtags 18-23 / wochende 12-24) und anderswo\",\"server_name\":\"coloRadio\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.coloradio.org\",\"stream_start\":\"Thu, 09 Feb 2023 03:35:11 +0100\",\"stream_start_iso8601\":\"2023-02-09T03:35:11+0100\",\"title\":\" - Es lÃ¤uft: Listen2Radio 2018 von JUNGES RADIO\",\"dummy\":null},{\"audio_bitrate\":256000,\"audio_channels\":2,\"audio_info\":\"samplerate=44100;channels=2;quality=8%2e00\",\"audio_samplerate\":44100,\"channels\":2,\"genre\":\"freies Radio\",\"ice-bitrate\":256,\"listener_peak\":3,\"listeners\":2,\"listenurl\":\"http://streaming.fueralle.org:80/coloradio_hq\",\"quality\":8.00,\"samplerate\":44100,\"server_description\":\"coloRadio, sendet auf 98,4 und 99,3 Mhz in Dresden, \",\"server_name\":\"coloRadio\",\"server_type\":\"application/ogg\",\"server_url\":\"http://coloradio.org\",\"stream_start\":\"Thu, 09 Feb 2023 03:35:03 +0100\",\"stream_start_iso8601\":\"2023-02-09T03:35:03+0100\",\"subtype\":\"Vorbis\",\"title\":\"Es läuft: Listen2Radio 2018 von JUNGES RADIO\",\"dummy\":null},{\"listeners\":0,\"listenurl\":\"http://streaming.fueralle.org:80/corax.mp3\",\"dummy\":null},{\"genre\":\"Radio CORAX\",\"listener_peak\":26,\"listeners\":15,\"listenurl\":\"http://streaming.fueralle.org:80/corax_128.mp3\",\"server_description\":\"Radio CORAX\",\"server_name\":\"Radio CORAX\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.radiocorax.de\",\"stream_start\":\"Tue, 07 Feb 2023 01:18:57 +0100\",\"stream_start_iso8601\":\"2023-02-07T01:18:57+0100\",\"dummy\":null},{\"genre\":\"Radio CORAX\",\"listener_peak\":25,\"listeners\":7,\"listenurl\":\"http://streaming.fueralle.org:80/corax_192.mp3\",\"server_description\":\"Radio CORAX\",\"server_name\":\"Radio CORAX\",\"server_type\":\"audio/mpeg\",\"server_url\":\"https://www.radiocorax.de\",\"stream_start\":\"Tue, 07 Feb 2023 01:18:57 +0100\",\"stream_start_iso8601\":\"2023-02-07T01:18:57+0100\",\"dummy\":null},{\"audio_info\":\"bitrate=256\",\"bitrate\":256,\"genre\":\"Radio CORAX\",\"listener_peak\":5,\"listeners\":0,\"listenurl\":\"http://streaming.fueralle.org:80/corax_256.mp3\",\"server_description\":\"Radio CORAX\",\"server_name\":\"Radio CORAX\",\"server_type\":\"audio/mpeg\",\"server_url\":\"https://radiocorax.de\",\"stream_start\":\"Tue, 07 Feb 2023 01:18:46 +0100\",\"stream_start_iso8601\":\"2023-02-07T01:18:46+0100\",\"dummy\":null},{\"genre\":\"Radio CORAX\",\"listener_peak\":8,\"listeners\":3,\"listenurl\":\"http://streaming.fueralle.org:80/corax_64.mp3\",\"server_description\":\"Radio CORAX\",\"server_name\":\"Radio CORAX\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.radiocorax.de\",\"stream_start\":\"Tue, 07 Feb 2023 01:18:57 +0100\",\"stream_start_iso8601\":\"2023-02-07T01:18:57+0100\",\"dummy\":null},{\"bitrate\":192,\"genre\":\"Freies Radio\",\"listener_peak\":11,\"listeners\":4,\"listenurl\":\"http://streaming.fueralle.org:80/frn\",\"server_description\":\"Freies Radio für Neumünster und Umgebung\",\"server_name\":\"Freies Radio Neumünster\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.freiesradio-nms.de\",\"stream_start\":\"Wed, 08 Feb 2023 18:37:03 +0100\",\"stream_start_iso8601\":\"2023-02-08T18:37:03+0100\",\"title\":null,\"dummy\":null},{\"audio_info\":\"ice-bitrate=128;ice-channels=2;ice-samplerate=44100\",\"genre\":\"Freie Radios\",\"ice-bitrate\":128,\"ice-channels\":2,\"ice-samplerate\":44100,\"listener_peak\":118,\"listeners\":21,\"listenurl\":\"http://streaming.fueralle.org:80/frs-hi.mp3\",\"server_description\":\"Freies Radio fuer Stuttgart\",\"server_name\":\"FRS-Webstream-Hi\",\"server_type\":\"audio/mpeg\",\"server_url\":\"www.freies-radio.de\",\"stream_start\":\"Tue, 24 Jan 2023 16:19:04 +0100\",\"stream_start_iso8601\":\"2023-01-24T16:19:04+0100\",\"dummy\":null},{\"audio_info\":\"ice-bitrate=56;ice-channels=2;ice-samplerate=44100\",\"genre\":\"Freie Radios\",\"ice-bitrate\":56,\"ice-channels\":2,\"ice-samplerate\":44100,\"listener_peak\":3,\"listeners\":0,\"listenurl\":\"http://streaming.fueralle.org:80/frs-lo.mp3\",\"server_description\":\"Freies Radio fuer Stuttgart (56kbps)\",\"server_name\":\"FRS-Webstream-Lo\",\"server_type\":\"audio/mpeg\",\"server_url\":\"www.freies-radio.de\",\"stream_start\":\"Wed, 01 Feb 2023 19:13:38 +0100\",\"stream_start_iso8601\":\"2023-02-01T19:13:38+0100\",\"dummy\":null},{\"bitrate\":128,\"genre\":\"Bildung und Unterhaltung\",\"listener_peak\":79,\"listeners\":5,\"listenurl\":\"http://streaming.fueralle.org:80/ginseng.mp3\",\"server_description\":\"Radio Ã60\",\"server_name\":\"RadioGinseng\",\"server_type\":\"audio/mpeg\",\"stream_start\":\"Thu, 02 Feb 2023 04:17:51 +0100\",\"stream_start_iso8601\":\"2023-02-02T04:17:51+0100\",\"dummy\":null},{\"genre\":\"Freies Radio Chemnitz\",\"listener_peak\":4,\"listeners\":1,\"listenurl\":\"http://streaming.fueralle.org:80/radiot.mp3\",\"server_description\":\"Radio T Chemnitz\",\"server_name\":\"Radio T Chemnitz\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.radiot-chemnitz.de\",\"stream_start\":\"Sat, 04 Feb 2023 01:24:28 +0100\",\"stream_start_iso8601\":\"2023-02-04T01:24:28+0100\",\"dummy\":null},{\"genre\":\"Freies Radio Chemnitz\",\"listener_peak\":10,\"listeners\":5,\"listenurl\":\"http://streaming.fueralle.org:80/radiot.ogg\",\"server_description\":\"Radio T Chemnitz\",\"server_name\":\"Radio T Chemnitz\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.radiot-chemnitz.de\",\"stream_start\":\"Sat, 04 Feb 2023 01:24:28 +0100\",\"stream_start_iso8601\":\"2023-02-04T01:24:28+0100\",\"dummy\":null},{\"genre\":\"Freies Radio Chemnitz\",\"listener_peak\":6,\"listeners\":0,\"listenurl\":\"http://streaming.fueralle.org:80/radiot_128.mp3\",\"server_description\":\"Radio T Chemnitz\",\"server_name\":\"Radio T Chemnitz\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.radiot-chemnitz.de\",\"stream_start\":\"Sat, 04 Feb 2023 01:22:18 +0100\",\"stream_start_iso8601\":\"2023-02-04T01:22:18+0100\",\"dummy\":null},{\"genre\":\"Freies Radio Chemnitz\",\"listener_peak\":3,\"listeners\":0,\"listenurl\":\"http://streaming.fueralle.org:80/radiot_192.mp3\",\"server_description\":\"Radio T Chemnitz\",\"server_name\":\"Radio T Chemnitz\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.radiot-chemnitz.de\",\"stream_start\":\"Sat, 04 Feb 2023 01:22:18 +0100\",\"stream_start_iso8601\":\"2023-02-04T01:22:18+0100\",\"dummy\":null},{\"audio_info\":\"bitrate=256\",\"bitrate\":256,\"genre\":\"Freies Radio Chemnitz\",\"listener_peak\":7,\"listeners\":2,\"listenurl\":\"http://streaming.fueralle.org:80/radiot_256.mp3\",\"server_description\":\"Radio T Chemnitz\",\"server_name\":\"Radio T Chemnitz\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.radiot-chemnitz.de/\",\"stream_start\":\"Sat, 04 Feb 2023 01:22:09 +0100\",\"stream_start_iso8601\":\"2023-02-04T01:22:09+0100\",\"dummy\":null},{\"genre\":\"Freies Radio Chemnitz\",\"listener_peak\":3,\"listeners\":0,\"listenurl\":\"http://streaming.fueralle.org:80/radiot_64.mp3\",\"server_description\":\"Radio T Chemnitz\",\"server_name\":\"Radio T Chemnitz\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.radiot-chemnitz.de\",\"stream_start\":\"Sat, 04 Feb 2023 01:22:18 +0100\",\"stream_start_iso8601\":\"2023-02-04T01:22:18+0100\",\"dummy\":null},{\"bitrate\":192,\"genre\":\"various\",\"listener_peak\":2,\"listeners\":2,\"listenurl\":\"http://streaming.fueralle.org:80/radiot_unterwegs\",\"server_description\":\"Unspecified description\",\"server_name\":\"Unspecified name\",\"server_type\":\"audio/mpeg\",\"stream_start\":\"Thu, 09 Feb 2023 17:41:35 +0100\",\"stream_start_iso8601\":\"2023-02-09T17:41:35+0100\",\"title\":\"Sitcom Warriors - i`ve been waiting for this a l\",\"dummy\":null},{\"audio_info\":\"channels=2;samplerate=44100;bitrate=128\",\"channels\":2,\"genre\":\"Community - Radio\",\"listener_peak\":10,\"listeners\":7,\"listenurl\":\"http://streaming.fueralle.org:80/slubfurt\",\"samplerate\":44100,\"server_description\":\"Freies BÃ¼rgerRadio Slubfurt\",\"server_name\":\"Freies BÃ¼rgerRadio Slubfurt Stream\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.radio.slubfurt.net/\",\"stream_start\":\"Tue, 07 Feb 2023 06:35:52 +0100\",\"stream_start_iso8601\":\"2023-02-07T06:35:52+0100\",\"title\":\"MUSIKZIRKUS - Die Nummer 20\",\"dummy\":null},{\"bitrate\":256,\"genre\":\"various\",\"listener_peak\":1,\"listeners\":0,\"listenurl\":\"http://streaming.fueralle.org:80/slubfurt256\",\"server_description\":\"Unspecified description\",\"server_name\":\"Unspecified name\",\"server_type\":\"audio/mpeg\",\"stream_start\":\"Sat, 04 Feb 2023 19:53:04 +0100\",\"stream_start_iso8601\":\"2023-02-04T19:53:04+0100\",\"title\":\"Honeybus - Can't Let Maggie Go\",\"dummy\":null},{\"genre\":\"Freies Radio Chemnitz\",\"listener_peak\":0,\"listeners\":0,\"listenurl\":\"http://streaming.fueralle.org:80/slubfurt_128.mp3\",\"server_description\":\"Freies BÃ¼rgerradio Slubfurt\",\"server_name\":\"Freies BÃ¼rgerradio Slubfurt\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://wwwradio.slubfurt.net\",\"stream_start\":\"Tue, 07 Feb 2023 06:35:56 +0100\",\"stream_start_iso8601\":\"2023-02-07T06:35:56+0100\",\"dummy\":null},{\"audio_info\":\"channels=2;samplerate=44100;bitrate=192\",\"bitrate\":192,\"channels\":2,\"genre\":\"Community - Radio\",\"listener_peak\":0,\"listeners\":0,\"listenurl\":\"http://streaming.fueralle.org:80/slubfurt_256\",\"samplerate\":44100,\"server_description\":\"Freies BÃ¼rgerRadio Slubfurt\",\"server_name\":\"Freies BÃ¼rgerRadio Slubfurt\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://www.radio.slubfurt.net/\",\"stream_start\":\"Tue, 07 Feb 2023 06:35:56 +0100\",\"stream_start_iso8601\":\"2023-02-07T06:35:56+0100\",\"title\":\"MUSIKZIRKUS - Die Nummer 20\",\"dummy\":null},{\"genre\":\"Freies Radio Chemnitz\",\"listener_peak\":0,\"listeners\":0,\"listenurl\":\"http://streaming.fueralle.org:80/slubfurt_64.mp3\",\"server_description\":\"Freies BÃ¼rgerradio Slubfurt\",\"server_name\":\"Freies BÃ¼rgerradio Slubfurt\",\"server_type\":\"audio/mpeg\",\"server_url\":\"http://wwwradio.slubfurt.net\",\"stream_start\":\"Tue, 07 Feb 2023 06:35:56 +0100\",\"stream_start_iso8601\":\"2023-02-07T06:35:56+0100\",\"dummy\":null}]}}"
	service      DefaultStreamScrapeService
)

func setupTest(t *testing.T) func() {
	recorder = httptest.NewRecorder()
	return func() {
	}
}

func Test_GetDatafromUrl_NoUrl_ReturnsError(t *testing.T) {
	teardown := setupTest(t)
	defer teardown()

	body, err := GetDataFromStreamUrl("")

	assert.Nil(t, body)
	assert.NotNil(t, err)
	assert.EqualValues(t, "Get \"\": unsupported protocol scheme \"\"", err.Error())
}

func Test_GetDatafromUrl_ReturnsNoError(t *testing.T) {
	teardown := setupTest(t)
	defer teardown()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, responseBody)
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

func Test_NewStreamScrapeService_Init(t *testing.T) {
	var cfg config.AppConfig
	service := NewStreamScrapeService(&cfg)

	assert.EqualValues(t, cfg, *service.Cfg)
}

func Test_Scrape_NoUrl_DoesntScrape(t *testing.T) {
	var cfg config.AppConfig
	service = NewStreamScrapeService(&cfg)
	service.Scrape()

	assert.EqualValues(t, false, cfg.RunTime.RunScrape)
}

func Test_ScrapeRun_LocalUrl_UpdatesMetrics(t *testing.T) {
	var cfg config.AppConfig

	teardown := setupTest(t)
	defer teardown()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, responseBody)
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

	service = NewStreamScrapeService(&cfg)
	ScrapeRun(service)

	assert.EqualValues(t, 1, cfg.RunTime.StreamScrapeCount)
}
