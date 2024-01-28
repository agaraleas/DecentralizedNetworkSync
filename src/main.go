package main

import (
	"fmt"
	"os"

	"github.com/agaraleas/DecentralizedNetworkSync/config"
	"github.com/agaraleas/DecentralizedNetworkSync/logging"
)

func main() {
	logging.InitLogging()

	cmdLineParseResult := parseCommandLineArgs()
	if cmdLineParseResult != nil {
		fmt.Println(cmdLineParseResult.msg)
		os.Exit(int(cmdLineParseResult.code))
	}

	fmt.Printf("Configured to connect to driver %s:%d with ticket %s\n",
		config.GlobalConfig.DriverInfo.Address.Host,
		config.GlobalConfig.DriverInfo.Address.Port,
		config.GlobalConfig.DriverInfo.Ticket)
}
