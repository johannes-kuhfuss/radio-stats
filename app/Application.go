package app

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/radio-stats/handlers"
	"github.com/johannes-kuhfuss/radio-stats/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/johannes-kuhfuss/services_utils/date"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

var (
	cfg                    config.AppConfig
	server                 http.Server
	appEnd                 chan os.Signal
	ctx                    context.Context
	cancel                 context.CancelFunc
	statsUiHandler         handlers.StatsUiHandler
	gpioSwitchHandler      handlers.GpioSwitchHandler
	streamScrapeService    service.DefaultStreamScrapeService
	gpioPollService        service.DefaultGpioPollService
	gpioSwitchService      service.DefaultGpioSwitchService
	streamVolDetectService service.StreamVolDetectService
	emberPollService       service.DefaultEmberPollService
)

func StartApp() {
	logger.Info("Starting application")

	getCmdLine()
	err := config.InitConfig(config.EnvFile, &cfg)
	if err != nil {
		panic(err)
	}
	initRouter()
	initServer()
	initMetrics()
	wireApp()
	mapUrls()
	RegisterForOsSignals()

	go startServer()
	go startStreamScraping()
	go startGpioPolling()
	go startEmberPolling()
	go startStreamVolumeDetect()

	<-appEnd
	cleanUp()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Graceful shutdown failed", err)
	} else {
		logger.Info("Graceful shutdown finished")
	}
}

func getCmdLine() {
	flag.StringVar(&config.EnvFile, "config.file", ".env", "Specify location of config file. Default is .env")
	flag.Parse()
}

func initRouter() {
	gin.SetMode(cfg.Gin.Mode)
	router := gin.New()
	if cfg.Gin.LogToLogger {
		gin.DefaultWriter = logger.GetLogger()
		router.Use(gin.Logger())
	}
	router.Use(gin.Recovery())
	router.SetTrustedProxies(nil)
	globPath := filepath.Join(cfg.Gin.TemplatePath, "*.tmpl")
	router.LoadHTMLGlob(globPath)

	cfg.RunTime.Router = router
}

func initServer() {
	var tlsConfig tls.Config

	if cfg.Server.UseTls {
		tlsConfig = tls.Config{
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
			PreferServerCipherSuites: true,
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		}
	}
	if cfg.Server.UseTls {
		cfg.RunTime.ListenAddr = fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.TlsPort)
	} else {
		cfg.RunTime.ListenAddr = fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	}

	server = http.Server{
		Addr:              cfg.RunTime.ListenAddr,
		Handler:           cfg.RunTime.Router,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 0,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    0,
	}
	if cfg.Server.UseTls {
		server.TLSConfig = &tlsConfig
		server.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	}
}

func wireApp() {
	statsUiHandler = handlers.NewStatsUiHandler(&cfg)
	streamScrapeService = service.NewStreamScrapeService(&cfg)
	gpioPollService = service.NewGpioPollService(&cfg)
	gpioSwitchService = service.NewGpioSwitchService(&cfg)
	gpioSwitchHandler = handlers.NewGpioSwitchHandler(&cfg, gpioSwitchService)
	streamVolDetectService = service.NewStreamVolDetectService(&cfg)
	emberPollService = service.NewEmberPollService(&cfg)
}

func mapUrls() {
	cfg.RunTime.Router.GET("/", statsUiHandler.StatusPage)
	cfg.RunTime.Router.GET("/about", statsUiHandler.AboutPage)
	cfg.RunTime.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	authorized := cfg.RunTime.Router.Group("/", basicAuth(cfg.Server.AdminUserName, cfg.Server.AdminPasswordHash))
	authorized.GET("/switch", statsUiHandler.SwitchPage)
	cfg.RunTime.Router.POST("/switch", gpioSwitchHandler.SwitchXpoint)
	cfg.RunTime.Router.GET("/logs", statsUiHandler.LogsPage)
}

func RegisterForOsSignals() {
	appEnd = make(chan os.Signal, 1)
	signal.Notify(appEnd, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
}

func startServer() {
	logger.Info(fmt.Sprintf("Listening on %v", cfg.RunTime.ListenAddr))
	cfg.RunTime.StartDate = date.GetNowUtc()
	if cfg.Server.UseTls {
		if err := server.ListenAndServeTLS(cfg.Server.CertFile, cfg.Server.KeyFile); err != nil && err != http.ErrServerClosed {
			logger.Error("Error while starting https server", err)
			panic(err)
		}
	} else {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Error while starting http server", err)
			panic(err)
		}
	}
}

func startStreamScraping() {
	streamScrapeService.Scrape()
}

func startGpioPolling() {
	gpioPollService.Poll()
}

func startEmberPolling() {
	emberPollService.Poll()
}

func startStreamVolumeDetect() {
	streamVolDetectService.Listen()
}

func cleanUp() {
	shutdownTime := time.Duration(cfg.Server.GracefulShutdownTime) * time.Second
	cfg.RunTime.RunListen = false
	cfg.RunTime.RunScrape = false
	cfg.RunTime.RunGpioPoll = false
	cfg.RunTime.RunEmberPoll = false
	emberPollService.CloseEmberConn()
	ctx, cancel = context.WithTimeout(context.Background(), shutdownTime)
	defer func() {
		logger.Info("Cleaning up")
		logger.Info("Done cleaning up")
		cancel()
	}()
}
