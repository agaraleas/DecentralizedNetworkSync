package config

var GlobalConfig AppConfig

type AppConfig struct {
	DriverUrl       string
	SharedDirectory string
}
