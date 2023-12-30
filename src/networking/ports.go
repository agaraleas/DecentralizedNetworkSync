package networking

import (
	"fmt"
	"net"

	"github.com/agaraleas/DecentralizedNetworkSync/logging"
)

const HighestAvailablePort = 65535

type Port uint16

func IsPortFree(port Port) bool {
	listeningAddress := ":" + fmt.Sprint(port)
	listener, err := net.Listen("tcp", listeningAddress)
	if err != nil {
		return false
	}

	listener.Close()
	return true
}

func FindFreePort() (Port, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		logging.Log.Error("Could not get an available port")
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return Port(addr.Port), nil
}