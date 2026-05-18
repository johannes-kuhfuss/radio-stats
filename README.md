# radio-stats

`radio-stats` collects operational metrics for radio streams, GPIO devices, Ember providers, and stream volume detection, then exposes status pages and Prometheus metrics through a Gin web server.

## Run

Create a `.env` file or pass one explicitly:

```powershell
go run . -config.file .env
```

Useful environment variables:

```text
SERVER_HOST=127.0.0.1
SERVER_PORT=8080
GIN_MODE=release
STREAM_SCRAPE_URL=http://example.test/status-json.xsl
STREAM_SCRAPE_INTERVAL_SEC=5
EXPECTED_SERVER_NAME=coloRadio
STREAM_VOLDETECT_URLS=http://example.test/stream.mp3
STREAM_VOLDETECT_FFMPEG=/usr/bin/ffmpeg
GPIO_HOST=192.0.2.10
GPIO_USER=reader
GPIO_PASSWORD=reader
GPIO_IN_CONFIG=1={"name":"Studio Alarm","invert":true}
GPIO_OUT_CONFIG=Studio=1
EMBER_IN_CONFIG=host={"port":9000,"entrypath":"1.2.3","metricsprefix":"ember_","gpios":["1","2"]}
ADMIN_USER_NAME=admin
ADMIN_PASSWORD_HASH=<bcrypt hash>
```

## Test

Run the full suite without cached results:

```powershell
go test ./... -count=1
```

Run with coverage:

```powershell
go test ./... -count=1 -coverprofile=coverage
go tool cover -func=coverage
```

On Windows, if the default Go build cache has permission problems, use a workspace-local cache:

```powershell
$env:GOCACHE = Join-Path (Get-Location) ".gocache"
go test ./... -count=1
```

## Notes

The long-running pollers support context cancellation internally. Tests use injected HTTP clients, ffmpeg runners, and Ember connections so they do not depend on live devices or streams.
