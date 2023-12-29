package networking

import (
	"net"
)

// dialer interface to abstract net.Dial and make it more testable
type dialer interface {
	Dial(network, address string) (net.Conn, error)
}

func getLocalIpThroughDialing(d dialer) (net.IP, error) {
	conn, err := d.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddress, ok := conn.LocalAddr().(*net.UDPAddr)
    if !ok {
        return nil, &net.AddrError{}
    }

	return localAddress.IP, nil
}

func GetLocalIP() net.IP {
	var dialer net.Dialer
    ip, err := getLocalIpThroughDialing(&dialer)

    if err != nil {
        panic(err)
    }

    return ip
}