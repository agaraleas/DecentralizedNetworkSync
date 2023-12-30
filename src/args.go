package main

import (
	"flag"
	"fmt"

	"github.com/agaraleas/DecentralizedNetworkSync/config"
	"github.com/agaraleas/DecentralizedNetworkSync/logging"
	"github.com/agaraleas/DecentralizedNetworkSync/networking"
)

//Interface for generic Command Line Argument handling
type CmdLineArgError struct {
	msg string
	code ReturnCode
}

type CmdLineArg interface {
	register()
	handle() *CmdLineArgError
}

func parseCommandLineArgs() *CmdLineArgError {
	args := gatherCommandLineArgTemplates()

	for _, arg := range(args){
		arg.register()
	}

	flag.Parse()

	return handleCommandLineArgValues(args)
}

func handleCommandLineArgValues(args []CmdLineArg) *CmdLineArgError {
	for _, arg := range(args){
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
	argTemplates = append(argTemplates, &portCmdLineArg{})
	return argTemplates
}

//Cmd Line Arg: Help
type helpCmdLineArg struct{
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

//Cmd Line Arg: Port
type portCmdLineArg struct{
	port int
}

func (arg *portCmdLineArg) register() {
	flag.IntVar(&arg.port, "p", 0, "Port number")
	flag.IntVar(&arg.port, "port", 0, "Port number")
}

func (arg *portCmdLineArg) handle() *CmdLineArgError {
	if !parseListeningPort(arg.port) {
		logging.Log.Error("Failed to parse listening port")
		errMsg := "Cannot listen to given port: " + fmt.Sprint(arg.port)
		return &CmdLineArgError{msg: errMsg, code: CantListenToPortError}
	}
	return nil
}

func parseListeningPort(suggestedPortNum int) bool {
	if suggestedPortNum < 0 || suggestedPortNum > networking.HighestAvailablePort {
		return false
	}

	if suggestedPortNum > 0 {
		portToTest := networking.Port(suggestedPortNum)
		if !networking.IsPortFree(portToTest){
			logging.Log.Warningf("Port parsing failed. Port %d is not free", portToTest)
			return false
		}
	}

	port := networking.Port(suggestedPortNum)
	if port == 0 {
		var err error
		if port, err = networking.FindFreePort(); err != nil {
			logging.Log.Warning("Port parsing failed. Could not find free port")
			return false
		}
	}

	logging.Log.Debugf("Setting application port to %d", port)
	config.GlobalConfig.Port = port
	return true
}








