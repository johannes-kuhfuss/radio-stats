# Build container
FROM golang:1.27.3-alpine
RUN apk -U upgrade --no-cache && apk add --no-cache go git && rm -rf /var/cache/apk/* && mkdir /build
WORKDIR /build
RUN git clone https://github.com/johannes-kuhfuss/radio-stats.git
WORKDIR /build/radio-stats
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /build/radio-stats/radio-stats /build/radio-stats/main.go
# Run container
FROM scratch
WORKDIR /app
COPY --from=0 /build/radio-stats/radio-stats /app/radio-stats
COPY --from=0 /build/radio-stats/templates /app/templates
COPY ffmpeg /usr/bin/ffmpeg
USER 101:101
HEALTHCHECK --interval=120s --timeout=5s CMD wget -q --spider http://localhost:8080/ || exit 1
ENTRYPOINT ["/app/radio-stats", "--config.file=/app/radio-stats.config"]