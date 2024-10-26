// package handlers sets up the handlers for the Web UI
package handlers

import (
	"fmt"
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

// NewGpioSwitchHandler sets up a new handler and injects its dependencies
func NewGpioSwitchHandler(cfg *config.AppConfig, svc service.GpioSwitchService) GpioSwitchHandler {
	return GpioSwitchHandler{
		Cfg: cfg,
		Svc: svc,
	}
}

// SwitchXpoint executes a switch on a crosspoint via GPIO
func (sh *GpioSwitchHandler) SwitchXpoint(c *gin.Context) {
	var switchReq dto.GpioSwitchRequest
	switchReq.Xpoint = c.PostForm("xpoint")
	if err := sh.validateReq(switchReq); err != nil {
		logger.Error("Error, no such xpoint", err)
		c.JSON(err.StatusCode(), err)
		return
	}
	if err := sh.Svc.Switch(switchReq.Xpoint); err != nil {
		logger.Error("Error while switching xpoint", err)
		c.JSON(err.StatusCode(), err)
		return
	}
	c.JSON(http.StatusOK, switchReq)
}

// validateReq makes sure the crosspoint (name) exists
func (sh *GpioSwitchHandler) validateReq(req dto.GpioSwitchRequest) api_error.ApiErr {
	_, ok := sh.Cfg.Gpio.OutConfig[req.Xpoint]
	if ok {
		return nil
	}
	return api_error.NewBadRequestError(fmt.Sprintf("xpoint with name %v does not exist", req.Xpoint))
}
