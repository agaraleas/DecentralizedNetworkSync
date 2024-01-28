package config

import (
	"github.com/agaraleas/DecentralizedNetworkSync/driver"
	"github.com/agaraleas/DecentralizedNetworkSync/networking"
)

var GlobalConfig AppConfig

type AppConfig struct {
	DriverInfo      driver.DriverConnectInfo
	ListenAddress   networking.Server
	SharedDirectory string
}
