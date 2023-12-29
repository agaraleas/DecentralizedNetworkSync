package main

import (
	"fmt"
	"net"

	"github.com/agaraleas/DecentralizedNetworkSync/networking"
)

func main() {
	dummyServer := networking.Server{Host: net.IPv4(127, 0, 0, 1), Port: 0}
	fmt.Printf("Hello from %s", dummyServer.ListeningAddress())
}