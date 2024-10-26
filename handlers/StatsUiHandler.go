// package handlers sets up the handlers for the Web UI
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/radio-stats/dto"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type StatsUiHandler struct {
	Cfg *config.AppConfig
}

// NewStatsUiHandler sets up a new handler and injects its dependencies
func NewStatsUiHandler(cfg *config.AppConfig) StatsUiHandler {
	return StatsUiHandler{
		Cfg: cfg,
	}
}

// StatusPage is the handler for the status page
func (uh *StatsUiHandler) StatusPage(c *gin.Context) {
	configData := dto.GetConfig(uh.Cfg)
	c.HTML(http.StatusOK, "status.page.tmpl", gin.H{
		"configdata": configData,
	})
}

// SwitchPage is the handler for the crosspoint switch page
func (uh *StatsUiHandler) SwitchPage(c *gin.Context) {
	configData := dto.GetConfig(uh.Cfg)
	c.HTML(http.StatusOK, "switch.page.tmpl", gin.H{
		"configdata": configData,
	})
}

// LogsPage is the handler for the page displaying log messages
func (uh *StatsUiHandler) LogsPage(c *gin.Context) {
	logs := logger.GetLogList()
	c.HTML(http.StatusOK, "logs.page.tmpl", gin.H{
		"title": "Logs",
		"logs":  logs,
	})
}

// AboutPage is the handler for the page displaying a short description of the program and its license
func (uh *StatsUiHandler) AboutPage(c *gin.Context) {
	c.HTML(http.StatusOK, "about.page.tmpl", nil)
}
