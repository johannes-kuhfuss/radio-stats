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

func NewStatsUiHandler(cfg *config.AppConfig) StatsUiHandler {
	return StatsUiHandler{
		Cfg: cfg,
	}
}

func (uh *StatsUiHandler) StatusPage(c *gin.Context) {
	configData := dto.GetConfig(uh.Cfg)
	c.HTML(http.StatusOK, "status.page.tmpl", gin.H{
		"configdata": configData,
	})
}

func (uh *StatsUiHandler) SwitchPage(c *gin.Context) {
	configData := dto.GetConfig(uh.Cfg)
	c.HTML(http.StatusOK, "switch.page.tmpl", gin.H{
		"configdata": configData,
	})
}

func (uh *StatsUiHandler) LogsPage(c *gin.Context) {
	logs := logger.GetLogList()
	c.HTML(http.StatusOK, "logs.page.tmpl", gin.H{
		"title": "Logs",
		"logs":  logs,
	})
}

func (uh *StatsUiHandler) AboutPage(c *gin.Context) {
	c.HTML(http.StatusOK, "about.page.tmpl", nil)
}
