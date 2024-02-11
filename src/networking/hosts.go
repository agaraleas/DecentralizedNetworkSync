package networking

import (
	"net"
	"strconv"

	"github.com/agaraleas/DecentralizedNetworkSync/logging"
)

type Server struct {
	Host net.IP
	Port Port
}

func (s *Server) ListeningAddress() string {
	return s.Host.String() + ":" + strconv.Itoa(int(s.Port))
}

func IsHostValid(host string) bool {
	ipAddr, err := net.ResolveIPAddr("ip", host)
	if err == nil {
		logging.Log.Debugf("Hostname %s (IP: %d) is valid", host, ipAddr.IP.To16())
		return true
	}

	ip := net.ParseIP(host)
	if ip != nil {
		logging.Log.Debugf("Host IP %s is valid", host)
		return true
	}

	logging.Log.Errorf("Host parsing failed. Host %s is not valid", host)
	return false
}
