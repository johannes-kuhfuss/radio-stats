package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/radio-stats/config"
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
	c.HTML(http.StatusOK, "status.page.tmpl", nil)
}

func (uh *StatsUiHandler) AboutPage(c *gin.Context) {
	c.HTML(http.StatusOK, "about.page.tmpl", nil)
}
