package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/agaraleas/DecentralizedNetworkSync/config"
	"github.com/agaraleas/DecentralizedNetworkSync/logging"
	"github.com/agaraleas/DecentralizedNetworkSync/networking"
)

// Interface for generic Command Line Argument handling
type CmdLineArgError struct {
	msg  string
	code ReturnCode
}

type CmdLineArg interface {
	register()
	handle() *CmdLineArgError
}

func parseCommandLineArgs() *CmdLineArgError {
	args := gatherCommandLineArgTemplates()

	for _, arg := range args {
		arg.register()
	}

	flag.Parse()

	return handleCommandLineArgValues(args)
}

func handleCommandLineArgValues(args []CmdLineArg) *CmdLineArgError {
	for _, arg := range args {
		err := arg.handle()

		if err != nil {
			return err
		}
	}

	return nil
}

func gatherCommandLineArgTemplates() []CmdLineArg {
	var argTemplates []CmdLineArg
	argTemplates = append(argTemplates, &helpCmdLineArg{})
	argTemplates = append(argTemplates, &sharedDirectoryCmdLineArg{})
	argTemplates = append(argTemplates, &driverCmdLineArg{})
	return argTemplates
}

// Cmd Line Arg: Help
type helpCmdLineArg struct {
	help bool
}

func (arg *helpCmdLineArg) register() {
	flag.BoolVar(&arg.help, "h", false, "Show help")
	flag.BoolVar(&arg.help, "help", false, "Show help")
}

func (arg *helpCmdLineArg) handle() *CmdLineArgError {
	if arg.help {
		helpMsg := createHelpMessage()
		return &CmdLineArgError{msg: helpMsg, code: NormalExit}
	}
	return nil
}

func createHelpMessage() string {
	msg := "==== DECENTRALIZED NETWORK SYNCER ====\n" +
		"Synchronizes access to network resources by coordinating\n" +
		"access requests with other active syncers in local network"
	return msg
}

// Cmd Line Arg: Shared Directory
type sharedDirectoryCmdLineArg struct {
	sharedDirectoryPath string
}

func (arg *sharedDirectoryCmdLineArg) register() {
	flag.StringVar(&arg.sharedDirectoryPath, "shared-dir", "", "Shared directory path for Syncers' communication")
}

func (arg *sharedDirectoryCmdLineArg) handle() *CmdLineArgError {
	if arg.sharedDirectoryPath == "" {
		errorMessage := fmt.Sprintf("Shared directory path cannot be empty")
		logging.Log.Error(errorMessage)
		return &CmdLineArgError{msg: errorMessage, code: InvalidSharedDirectoryError}
	}

	fileInfo, err := os.Stat(arg.sharedDirectoryPath)
	if err != nil {
		var errorMessage string
		if os.IsNotExist(err) {
			errorMessage = fmt.Sprintf("Shared directory path does not exist: %s", arg.sharedDirectoryPath)
		} else {
			errorMessage = fmt.Sprintf("Invalid shared directory path: %v", err)
		}
		logging.Log.Error(errorMessage)
		return &CmdLineArgError{msg: errorMessage, code: InvalidSharedDirectoryError}
	}

	if !fileInfo.IsDir() {
		errorMessage := fmt.Sprintf("Shared directory path does not point to a directory: %s", arg.sharedDirectoryPath)
		logging.Log.Error(errorMessage)
		return &CmdLineArgError{msg: errorMessage, code: InvalidSharedDirectoryError}
	}

	config.GlobalConfig.SharedDirectory = arg.sharedDirectoryPath
	return nil
}

// Cmd Line Arg: Driver
type driverCmdLineArg struct {
	host   string
	port   int
	ticket string
}

func (arg *driverCmdLineArg) register() {
	flag.StringVar(&arg.host, "driver-host", "localhost", "Address of host App which drives Syncer")
	flag.IntVar(&arg.port, "driver-port", 0, "Port which host app listens")
	flag.StringVar(&arg.ticket, "driver-ticket", "", "Ticket to connect to host app")
}

func (arg *driverCmdLineArg) handle() *CmdLineArgError {
	if !parseDriverHost(arg.host) {
		logging.Log.Error("Failed to parse driver host")
		errMsg := fmt.Sprintf("Error in parsing arg --driver-host. Invalid host: %s", arg.host)
		return &CmdLineArgError{msg: errMsg, code: InvalidHostError}
	}
	if !parseDriverPort(arg.port) {
		logging.Log.Error("Failed to parse driver port")
		errMsg := "Error in parsing arg --driver-port. Invalid port: " + fmt.Sprint(arg.port)
		return &CmdLineArgError{msg: errMsg, code: InvalidPortError}
	}
	config.GlobalConfig.DriverInfo.Ticket = arg.ticket
	return nil
}

func parseDriverPort(suggestedPortNum int) bool {
	if suggestedPortNum <= 0 || suggestedPortNum > networking.HighestAvailablePort {
		logging.Log.Errorf("Port parsing failed. Port %d is not valid", suggestedPortNum)
		return false
	}

	logging.Log.Debugf("Setting application port to %d", suggestedPortNum)
	config.GlobalConfig.DriverInfo.Address.Port = networking.Port(suggestedPortNum)
	return true
}

func parseDriverHost(host string) bool {
	ipAddr, err := net.ResolveIPAddr("ip", host)
	if err == nil {
		config.GlobalConfig.DriverInfo.Address.Host = ipAddr.IP.To16()
		return true
	}

	ip := net.ParseIP(host)
	if ip != nil {
		config.GlobalConfig.DriverInfo.Address.Host = ip
		return true
	}

	logging.Log.Errorf("Host parsing failed. Host %s is not valid", host)
	return false
}
