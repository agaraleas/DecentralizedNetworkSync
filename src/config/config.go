package config

import "github.com/agaraleas/DecentralizedNetworkSync/networking"

var GlobalConfig AppConfig

type AppConfig struct {
	Port networking.Port
}

