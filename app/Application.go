// package app ties together all bits and pieces to start the program
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

// StartApp orchestrates the startup of the application
func StartApp() {
	logger.Info("Starting application")

	getCmdLine()
	if err := config.InitConfig(config.EnvFile, &cfg); err != nil {
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

// getCmdLine checks the command line arguments
func getCmdLine() {
	flag.StringVar(&config.EnvFile, "config.file", ".env", "Specify location of config file. Default is .env")
	flag.Parse()
}

// initRouter initializes gin-gonic as the router
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

// initServer checks whether https is enabled and initializes the web server accordingly
func initServer() {
	var tlsConfig tls.Config

	if cfg.Server.UseTls {
		tlsConfig = tls.Config{
			PreferServerCipherSuites: true,
			MinVersion:               tls.VersionTLS13,
			CurvePreferences: []tls.CurveID{
				tls.X25519,
				tls.CurveP256,
				tls.CurveP384,
			},
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

// wireApp initializes the services in the right order and injects the dependencies
func wireApp() {
	statsUiHandler = handlers.NewStatsUiHandler(&cfg)
	streamScrapeService = service.NewStreamScrapeService(&cfg)
	gpioPollService = service.NewGpioPollService(&cfg)
	gpioSwitchService = service.NewGpioSwitchService(&cfg)
	gpioSwitchHandler = handlers.NewGpioSwitchHandler(&cfg, gpioSwitchService)
	streamVolDetectService = service.NewStreamVolDetectService(&cfg)
	emberPollService = service.NewEmberPollService(&cfg)
}

// mapUrls defines the handlers for the available URLs
func mapUrls() {
	cfg.RunTime.Router.GET("/", statsUiHandler.StatusPage)
	cfg.RunTime.Router.GET("/about", statsUiHandler.AboutPage)
	cfg.RunTime.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	authorized := cfg.RunTime.Router.Group("/", basicAuth(cfg.Server.AdminUserName, cfg.Server.AdminPasswordHash))
	authorized.GET("/switch", statsUiHandler.SwitchPage)
	cfg.RunTime.Router.POST("/switch", gpioSwitchHandler.SwitchXpoint)
	cfg.RunTime.Router.GET("/logs", statsUiHandler.LogsPage)
}

// RegisterForOsSignals listens for OS signals terminating the program and sends an internal signal to start cleanup
func RegisterForOsSignals() {
	appEnd = make(chan os.Signal, 1)
	signal.Notify(appEnd, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
}

// startServer starts the preconfigured web server
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

// startStreamScraping starts the stream scraping
func startStreamScraping() {
	streamScrapeService.Scrape()
}

// startGpioPolling starts the GPIO polling
func startGpioPolling() {
	gpioPollService.Poll()
}

// startEmberPolling starts the Ember polling
func startEmberPolling() {
	emberPollService.Poll()
}

// startStreamVolumeDetect starts the detections of stream volumes
func startStreamVolumeDetect() {
	streamVolDetectService.Listen()
}

// cleanUp tries to clean up when the program is stopped
func cleanUp() {
	logger.Info("Cleaning up")
	shutdownTime := time.Duration(cfg.Server.GracefulShutdownTime) * time.Second
	cfg.RunTime.RunListen = false
	cfg.RunTime.RunScrape = false
	cfg.RunTime.RunGpioPoll = false
	cfg.RunTime.RunEmberPoll = false
	emberPollService.CloseEmberConn()
	ctx, cancel = context.WithTimeout(context.Background(), shutdownTime)
	defer func() {
		logger.Info("Done cleaning up")
		cancel()
	}()
}
