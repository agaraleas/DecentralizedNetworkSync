package networking

import (
	"fmt"
	"net"
	"strconv"

	"github.com/agaraleas/DecentralizedNetworkSync/logging"
)

const HighestAvailablePort = 65535
const LowestAvailablePort = 0

type Port uint16

func IsPortValid(port Port) bool {
	if port <= LowestAvailablePort || port > HighestAvailablePort {
		logging.Log.Errorf("Port %d is not valid", port)
		return false
	}
	return true
}

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

func ToPort(str string) (Port, error) {
	num, err := strconv.ParseUint(str, 10, 16)
	if err != nil {
		logging.Log.Errorf("Error converting string to uint16: %v", err)
		return Port(0), err
	}

	return Port(num), nil
}
