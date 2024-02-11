package main

import (
	"fmt"
	"os"

	"github.com/agaraleas/DecentralizedNetworkSync/logging"
)

func main() {
	logging.InitLogging()

	cmdLineParseResult := parseCommandLineArgs()
	if cmdLineParseResult != nil {
		fmt.Println(cmdLineParseResult.msg)
		os.Exit(int(cmdLineParseResult.code))
	}
}
