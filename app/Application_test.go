package app

import (
	"crypto/tls"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/radio-stats/service"
	"github.com/stretchr/testify/assert"
)

func TestInitServerWithoutTlsSetsHttpAddress(t *testing.T) {
	cfg = config.AppConfig{}
	cfg.RunTime.Router = gin.New()
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = "8081"
	cfg.Server.UseTls = false

	initServer()

	assert.EqualValues(t, "127.0.0.1:8081", cfg.RunTime.ListenAddr)
	assert.EqualValues(t, cfg.RunTime.ListenAddr, server.Addr)
	assert.Nil(t, server.TLSConfig)
}

func TestInitServerWithTlsSetsTlsAddressAndConfig(t *testing.T) {
	cfg = config.AppConfig{}
	cfg.RunTime.Router = gin.New()
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.TlsPort = "8444"
	cfg.Server.UseTls = true

	initServer()

	assert.EqualValues(t, "127.0.0.1:8444", cfg.RunTime.ListenAddr)
	assert.EqualValues(t, cfg.RunTime.ListenAddr, server.Addr)
	assert.NotNil(t, server.TLSConfig)
	assert.EqualValues(t, tls.VersionTLS13, server.TLSConfig.MinVersion)
	assert.NotNil(t, server.TLSNextProto)
}

func TestCleanUpStopsRuntimeLoops(t *testing.T) {
	cfg = config.AppConfig{}
	cfg.Server.GracefulShutdownTime = 1
	cfg.SetRunListen(true)
	cfg.SetRunScrape(true)
	cfg.SetRunGpioPoll(true)
	cfg.SetRunEmberPoll(true)
	cfg.RunTime.EmberGpios = make(map[string]config.EmberConfig)
	emberPollService = service.NewEmberPollService(&cfg)

	cleanUp()

	assert.False(t, cfg.ShouldRunListen())
	assert.False(t, cfg.ShouldRunScrape())
	assert.False(t, cfg.ShouldRunGpioPoll())
	assert.False(t, cfg.ShouldRunEmberPoll())
	assert.NotNil(t, ctx)
}
