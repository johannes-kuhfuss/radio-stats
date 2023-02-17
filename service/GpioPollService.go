package service

import (
	"fmt"
	"time"

	"github.com/johannes-kuhfuss/radio-stats/config"
	"github.com/johannes-kuhfuss/services_utils/logger"

	"go.bug.st/serial"
)

type GpioPollService interface {
	Poll()
}

type DefaultGpioPollService struct {
	Cfg        *config.AppConfig
	serialPort serial.Port
}

func NewGpioPollService(cfg *config.AppConfig) DefaultGpioPollService {
	port := connectSerial(cfg.Gpio.SerialPort)
	return DefaultGpioPollService{
		Cfg:        cfg,
		serialPort: *port,
	}
}

func connectSerial(portName string) *serial.Port {
	mode := &serial.Mode{
		BaudRate: 9600,
	}
	port, err := serial.Open(portName, mode)
	if err != nil {
		logger.Error("Could not open serial port: ", err)
		return nil
	}
	return &port
}

func (s DefaultGpioPollService) Poll() {
	logger.Info(fmt.Sprintf("Starting to poll GPIOs on port %v", s.Cfg.Gpio.SerialPort))
	_, err := s.serialPort.Write([]byte("gpio read 0\n\r"))
	if err != nil {
		logger.Error("Error writing to serial port: ", err)
	}
	buff := make([]byte, 100)
	for {
		n, err := s.serialPort.Read(buff)
		if err != nil {
			logger.Error("Error reading from serial port: ", err)
			break
		}
		if n == 0 {
			logger.Info("Serial is EOF")
			break
		}
		logger.Info(fmt.Sprintf("Serial data: %v", string(buff[:n])))
	}
	for {
		time.Sleep(time.Duration(s.Cfg.Gpio.GpioPollIntervalSec) * time.Second)
	}
}
