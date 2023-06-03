package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/radio-stats/dto"
	"github.com/johannes-kuhfuss/radio-stats/service"
	"github.com/johannes-kuhfuss/services_utils/api_error"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type GpioSwitchHandler struct {
	Cfg *config.AppConfig
	Svc service.GpioSwitchService
}

func NewGpioSwitchHandler(cfg *config.AppConfig, svc service.GpioSwitchService) GpioSwitchHandler {
	return GpioSwitchHandler{
		Cfg: cfg,
		Svc: svc,
	}
}

func (sh *GpioSwitchHandler) SwitchXpoint(c *gin.Context) {
	var switchReq dto.GpioSwitchRequest
	if err := c.ShouldBindJSON(&switchReq); err != nil {
		msg := "Invalid JSON body in switch request"
		logger.Error(msg, err)
		apiErr := api_error.NewBadRequestError(msg)
		c.JSON(apiErr.StatusCode(), apiErr)
		return
	}
	// validate string!
	err := sh.Svc.Switch(switchReq.Xpoint)
	if err != nil {
		logger.Error("Error while switching xpoint", err)
		c.JSON(err.StatusCode(), err)
		return
	}
	c.JSON(http.StatusOK, nil)
}
