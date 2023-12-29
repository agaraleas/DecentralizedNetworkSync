package main

import (
	"fmt"

	"github.com/agaraleas/DecentralizedNetworkSync/networking"
)

func main() {
	ip := networking.GetLocalIP()
	dummyServer := networking.Server{Host: ip, Port: 0}
	fmt.Printf("Hello from %s", dummyServer.ListeningAddress())
}