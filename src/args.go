package main

import (
	"flag"
	"fmt"
	"net"
	"net/url"
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
	argTemplates = append(argTemplates, &driverUrlCmdLineArg{})
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

// Cmd Line Arg: Driver Url
type driverUrlCmdLineArg struct {
	websocketServerUrl string
}

func (arg *driverUrlCmdLineArg) register() {
	flag.StringVar(&arg.websocketServerUrl, "driver-url", "", "The websocket server url of the driver")
}

func (arg *driverUrlCmdLineArg) handle() *CmdLineArgError {
	u, err := url.Parse(arg.websocketServerUrl)
	if err != nil {
		errorMessage := fmt.Sprintf("Error parsing url: %v", err)
		logging.Log.Error(errorMessage)
		return &CmdLineArgError{msg: errorMessage, code: InvalidDriverUrlError}
	}

	if u.Scheme != "ws" && u.Scheme != "wss" {
		errorMessage := fmt.Sprintf("Websocket scheme should be 'ws' or 'wss': %v", err)
		logging.Log.Error(errorMessage)
		return &CmdLineArgError{msg: errorMessage, code: InvalidDriverUrlError}
	}

	if u.Host == "" {
		errorMessage := fmt.Sprintf("Unspecified host': %v", err)
		logging.Log.Error(errorMessage)
		return &CmdLineArgError{msg: errorMessage, code: InvalidDriverUrlError}
	}

	host, portValue, err := net.SplitHostPort(u.Host)
	if err != nil {
		errorMessage := fmt.Sprintf("Could not split url to host and port': %v", err)
		logging.Log.Error(errorMessage)
		return &CmdLineArgError{msg: errorMessage, code: InvalidDriverUrlError}
	}

	port, err := networking.ToPort(portValue)
	if err != nil {
		errorMessage := fmt.Sprintf("Could not convert %d to Port': %v", port, err)
		logging.Log.Error(errorMessage)
		return &CmdLineArgError{msg: errorMessage, code: InvalidDriverUrlError}
	}

	if !networking.IsPortValid(port) {
		errorMessage := fmt.Sprintf("Port %d is invalid", port)
		return &CmdLineArgError{msg: errorMessage, code: InvalidDriverUrlError}
	}

	if !networking.IsHostValid(host) {
		errorMessage := fmt.Sprintf("Host %s is invalid", host)
		return &CmdLineArgError{msg: errorMessage, code: InvalidDriverUrlError}
	}

	config.GlobalConfig.DriverUrl = arg.websocketServerUrl
	return nil
}
