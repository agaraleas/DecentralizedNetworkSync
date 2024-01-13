package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

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
	argTemplates = append(argTemplates, &sharedFolderCmdLineArg{})
	argTemplates = append(argTemplates, &driverCmdLineArg{})
	argTemplates = append(argTemplates, &listenCmdLineArg{})
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

// Cmd Line Arg: Shared Folder
type sharedFolderCmdLineArg struct {
	sharedFolderPath string
}

func (arg *sharedFolderCmdLineArg) register() {
	flag.StringVar(&arg.sharedFolderPath, "shared-folder", "", "Shared folder path for Syncers' communication")
}

func (arg *sharedFolderCmdLineArg) handle() *CmdLineArgError {
	if arg.sharedFolderPath == "" {
		errorMessage := fmt.Sprintf("Shared folder path cannot be empty")
		logging.Log.Error(errorMessage)
		return &CmdLineArgError{msg: errorMessage, code: InvalidSharedFolderError}
	}

	fileInfo, err := os.Stat(arg.sharedFolderPath)
	if err != nil {
		var errorMessage string
		if os.IsNotExist(err) {
			errorMessage = fmt.Sprintf("Shared folder path does not exist: %s", arg.sharedFolderPath)
		} else {
			errorMessage = fmt.Sprintf("Invalid shared folder path: %v", err)
		}
		logging.Log.Error(errorMessage)
		return &CmdLineArgError{msg: errorMessage, code: InvalidSharedFolderError}
	}

	if !fileInfo.IsDir() {
		errorMessage := fmt.Sprintf("Shared folder path does not point to a directory: %s", arg.sharedFolderPath)
		logging.Log.Error(errorMessage)
		return &CmdLineArgError{msg: errorMessage, code: InvalidSharedFolderError}
	}

	config.GlobalConfig.SharedFolder = arg.sharedFolderPath
	return nil
}

// Cmd Line Arg: Driver
type driverCmdLineArg struct {
	host   string
	port   int
	ticket string
}

func (arg *driverCmdLineArg) register() {
	flag.StringVar(&arg.host, "driver-host", "", "Address of host App which drives Syncer")
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
	ip := net.ParseIP(host)
	if ip == nil {
		logging.Log.Errorf("Host parsing failed. Host %s is not valid", host)
		return false
	}

	config.GlobalConfig.DriverInfo.Address.Host = ip
	return true
}

// Cmd Line Arg: listen
type listenCmdLineArg struct {
	address string
}

func (arg *listenCmdLineArg) register() {
	flag.StringVar(&arg.address, "listen", "", "Address which Syncer listens")
}

func (arg *listenCmdLineArg) handle() *CmdLineArgError {
	if arg.address == "" {
		logging.Log.Error("Listen parsing failed. Listen address not provided")
		return &CmdLineArgError{msg: "Error in arg --listen. Listen address not provided",
			code: CantListenToPortError}
	}

	addressComponents := strings.SplitN(arg.address, ":", 2)
	if len(addressComponents) != 2 {
		logging.Log.Errorf("Listen parsing failed. Address %s does not hold valid format x.x.x.x:ppppp", arg.address)
		return &CmdLineArgError{msg: "Error in parsing arg --listen. Invalid listen address",
			code: CantListenToPortError}
	}

	if addressComponents[0] == "" {
		addressComponents[0] = "127.0.0.1"
	}

	ip := net.ParseIP(addressComponents[0])
	if ip == nil {
		logging.Log.Errorf("Listening to %s failed. Host %s is not a valid IP", arg.address, addressComponents[0])
		return &CmdLineArgError{msg: "Error in parsing arg --listen. Invalid listen address",
			code: CantListenToPortError}
	}

	port, err := parseListenPort(addressComponents[1])
	if err != nil {
		logging.Log.Errorf("Listening to port %s failed. %s", addressComponents[1], err)
		return &CmdLineArgError{msg: fmt.Sprintf("Error in parsing arg --listen. %s", err.Error()),
			code: CantListenToPortError}
	}

	config.GlobalConfig.ListenAddress.Host = ip
	config.GlobalConfig.ListenAddress.Port = port
	return nil
}

func parseListenPort(portText string) (networking.Port, error) {
	portNum, err := strconv.Atoi(portText)
	if err != nil {
		return 0, fmt.Errorf("Invalid port number")
	}

	if portNum <= 0 || portNum > networking.HighestAvailablePort {
		return 0, fmt.Errorf("Invalid port number")
	}

	port := networking.Port(portNum)
	if !networking.IsPortFree(port) {
		return 0, fmt.Errorf("Port is not free")
	}

	logging.Log.Debugf("Setting application port to %d", port)
	return port, nil
}
