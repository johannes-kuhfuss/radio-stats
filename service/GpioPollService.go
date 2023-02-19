package service

import (
	"encoding/hex"
	"fmt"
	"regexp"
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
	serialPort *serial.Port
}

var (
	runPoll bool = false
)

func NewGpioPollService(cfg *config.AppConfig) DefaultGpioPollService {
	var port *serial.Port
	if cfg.Gpio.SerialPort != "" {
		port = connectSerial(cfg.Gpio.SerialPort)
		if port != nil {
			runPoll = true
			cfg.RunTime.SerialPort = *port
		}
	} else {
		logger.Warn("No serial port configured, not polling")
	}
	return DefaultGpioPollService{
		Cfg:        cfg,
		serialPort: port,
	}
}

func connectSerial(portName string) *serial.Port {
	mode := &serial.Mode{
		BaudRate: 19200,
	}
	port, err := serial.Open(portName, mode)
	if err != nil {
		logger.Error("Could not open serial port: ", err)
		return nil
	}
	return &port
}

func (s DefaultGpioPollService) Poll() {
	if runPoll == true {
		logger.Info(fmt.Sprintf("Starting to poll GPIOs on port %v", s.Cfg.Gpio.SerialPort))
		for {
			serialData := readFromSerial(*s.serialPort)
			stateHex := findHexVals(serialData)
			if len(stateHex) > 1 {
				stateBool := toBoolArray(stateHex[1])
				if len(stateBool) == 8 {
					setGpioState(s.Cfg, stateBool)
					updateGpioMetrics(s.Cfg, stateBool)
				}
			}
			time.Sleep(time.Duration(s.Cfg.Gpio.GpioPollIntervalSec) * time.Second)
		}
	}
}

func readFromSerial(port serial.Port) string {
	buff := make([]byte, 100)
	_, err := port.Write([]byte("gpio readall\n\r"))
	if err != nil {
		logger.Error("Error writing to serial port: ", err)
	}
	n, err := port.Read(buff)
	if err != nil {
		logger.Error("Error reading from serial port: ", err)
	}
	if n == 0 {
		logger.Info("Serial is EOF")
	}
	return string(buff[:n])
}

func findHexVals(serialData string) []byte {
	var stateHex []byte
	regEx, err := regexp.Compile("[A-F0-9]{4}")
	if err != nil {
		logger.Error("Error while compiling regex: ", err)
	}
	stateString := regEx.FindStringSubmatch(serialData)
	if len(stateString) > 0 {
		stateHex, err = hex.DecodeString(stateString[0])
		if err != nil {
			logger.Error("Error when converting to hexadecimal: ", err)
		}
	}
	return stateHex
}

func toBoolArray(number byte) []bool {
	result := []bool{}
	for number > 0 {
		result = append([]bool{number%2 == 1}, result...)
		number >>= 1
	}
	return result
}

func setGpioState(cfg *config.AppConfig, stateBool []bool) {
	cfg.RunTime.Gpio01State = !stateBool[7]
	cfg.RunTime.Gpio02State = !stateBool[6]
	cfg.RunTime.Gpio03State = !stateBool[5]
	cfg.RunTime.Gpio04State = !stateBool[4]
	cfg.RunTime.Gpio05State = !stateBool[3]
	cfg.RunTime.Gpio06State = !stateBool[2]
	cfg.RunTime.Gpio07State = !stateBool[1]
	cfg.RunTime.Gpio08State = !stateBool[0]
}

func updateGpioMetrics(cfg *config.AppConfig, stateBool []bool) {
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio01Name).Set(float64(boolToInt(!stateBool[7])))
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio02Name).Set(float64(boolToInt(!stateBool[6])))
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio03Name).Set(float64(boolToInt(!stateBool[5])))
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio04Name).Set(float64(boolToInt(!stateBool[4])))
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio05Name).Set(float64(boolToInt(!stateBool[3])))
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio06Name).Set(float64(boolToInt(!stateBool[2])))
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio07Name).Set(float64(boolToInt(!stateBool[1])))
	cfg.Metrics.GpioStateGauge.WithLabelValues(cfg.Gpio.Gpio08Name).Set(float64(boolToInt(!stateBool[0])))
}

func boolToInt(state bool) int {
	if state == true {
		return 1
	} else {
		return 0
	}
}
