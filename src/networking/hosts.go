package networking

import (
	"net"
	"strconv"
)

type Server struct {
	Host net.IP
	Port uint16
}

func (s* Server) ListeningAddress() string {
	return s.Host.String() + ":" + strconv.Itoa(int(s.Port))
}