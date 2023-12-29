package networking

import (
	"log"
	"net"
)

func getLocalIpThroughDialing(dialer *net.Dialer) net.IP {
	conn, err := dialer.Dial("udp", "8.8.8.8:80")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    localAddress := conn.LocalAddr().(*net.UDPAddr)
    return localAddress.IP
}

func GetLocalIP() net.IP {
	var dialer net.Dialer
    return getLocalIpThroughDialing(&dialer)
}