package driver

import "github.com/agaraleas/DecentralizedNetworkSync/networking"

type DriverConnectInfo struct {
	Address networking.Server
	Ticket string
}