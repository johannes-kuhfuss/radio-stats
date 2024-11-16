# Build container
FROM alpine:latest
RUN apk -U upgrade --no-cache && apk add --no-cache go git && rm -rf /var/cache/apk/* && mkdir /build
WORKDIR /build
RUN git clone https://github.com/johannes-kuhfuss/radio-stats.git
WORKDIR /build/radio-stats
RUN go build -o /build/radio-stats/radio-stats /build/radio-stats/main.go
# Run container
FROM alpine:latest
RUN apk -U upgrade --no-cache && apk add --no-cache ffmpeg && rm -rf /var/cache/apk/* && mkdir /app
WORKDIR /app
COPY --from=0 /build/radio-stats/radio-stats /app/radio-stats
COPY --from=0 /build/radio-stats/templates /app/templates
RUN addgroup -g 101 rstats && adduser -s /sbin/nologin -G rstats -D -H -u 101 rstats
USER rstats
HEALTHCHECK --interval=120s --timeout=5s CMD wget -q --spider http://localhost:8080/ || exit 1
ENTRYPOINT ["/app/radio-stats", "--config.file=/app/radio-stats.config"]
