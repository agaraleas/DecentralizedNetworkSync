package main

import (
	"flag"
	"fmt"
	"net"
	"testing"

	"github.com/agaraleas/DecentralizedNetworkSync/config"
	"github.com/agaraleas/DecentralizedNetworkSync/networking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatherCommandLineArgTemplates(t *testing.T) {
	args := gatherCommandLineArgTemplates()
	require.GreaterOrEqual(t, len(args), 2)

	index := 0
	_, hasCorrectType := args[index].(*helpCmdLineArg)
	assert.True(t, hasCorrectType, fmt.Sprintf("Index %d has unexpected type instead of helpCmdLineArg", index))

	index += 1
	_, hasCorrectType = args[index].(*portCmdLineArg)
	assert.True(t, hasCorrectType, fmt.Sprintf("Index %d has unexpected type instead of portCmdLineArg", index))
}

func TestHelpCmdLineArgRegistering(t *testing.T) {
	require.Nil(t, flag.Lookup("h"), "Cannot proceed in actual test of registering. -h already exists")
	require.Nil(t, flag.Lookup("help"), "Cannot proceed in actual test of registering. --help already exists")
	helpArg := helpCmdLineArg{}
	helpArg.register()
	assert.NotNil(t, flag.Lookup("h"), "Failed to register -h flag")
	assert.NotNil(t, flag.Lookup("help"), "Failed to register --help flag")
}
func TestHelpCmdLineArgHandling(t *testing.T) {
	helpAsked := helpCmdLineArg{help: true}
	outcome := helpAsked.handle()
	require.NotNil(t, outcome, "Hanlding of help cmd line arg returned nil")
	assert.Equal(t, outcome.msg, createHelpMessage(), "Hanlding of help cmd line arg misses help message")
	assert.Equal(t, outcome.code, NormalExit, "Hanlding of help cmd line arg has incorrect return code")

	helpNotAsked := helpCmdLineArg{help: false}
	outcome = helpNotAsked.handle()
	assert.Nil(t, outcome, "Hanlding of help cmd line arg returned error althogh help not requested")
}

func TestPortCmdLineArgRegistering(t *testing.T) {
	require.Nil(t, flag.Lookup("p"), "Cannot proceed in actual test of registering. -p already exists")
	require.Nil(t, flag.Lookup("port"), "Cannot proceed in actual test of registering. --port already exists")
	portArg := portCmdLineArg{}
	portArg.register()
	assert.NotNil(t, flag.Lookup("p"), "Failed to register -p flag")
	assert.NotNil(t, flag.Lookup("port"), "Failed to register --port flag")
}
func TestPortmdLineArgHandling(t *testing.T) {
	oldPort := config.GlobalConfig.Port
	revertPort := func() {
		config.GlobalConfig.Port = oldPort
	}
	defer revertPort()

	//Get a free port
	freePort, err := networking.FindFreePort()
	require.Nil(t, err, "Failed to get a free port to continue with actual test")

	//Test case a: provide a valid free port
	portProvided := portCmdLineArg{port: int(freePort)}
	outcome := portProvided.handle()
	assert.Nil(t, outcome, "Hanlding of port cmd line arg returned error")
	assert.Equal(t, config.GlobalConfig.Port, freePort, "Failed to assign requested port in global config")

	//Test case b: dont provide a port
	previousGlobalPort := config.GlobalConfig.Port
	portNotProvided := portCmdLineArg{}
	outcome = portNotProvided.handle()
	assert.Nil(t, outcome, "Hanlding of port cmd line arg returned error when port was not given")
	assert.NotEqual(t, portNotProvided.port, previousGlobalPort, "Failed to assign a random free port in global config")

	//Test case c: provide a negative port number
	previousGlobalPort = config.GlobalConfig.Port
	negativePortProvided :=  portCmdLineArg{port: -1}
	outcome = negativePortProvided.handle()
	assert.NotNil(t, outcome, "Hanlding of invalid port did not result in an error")
	assert.Equal(t, outcome.msg, "Cannot listen to given port: -1")
	assert.Equal(t, outcome.code, CantListenToPortError)
	assert.Equal(t, config.GlobalConfig.Port, previousGlobalPort)

	//Test case d: provide a higher port number than allowed
	previousGlobalPort = config.GlobalConfig.Port
	highPortProvided :=  portCmdLineArg{port: 1234567}
	outcome = highPortProvided.handle()
	assert.NotNil(t, outcome, "Hanlding of high port did not result in an error")
	assert.Equal(t, outcome.msg, "Cannot listen to given port: 1234567")
	assert.Equal(t, outcome.code, CantListenToPortError)
	assert.Equal(t, config.GlobalConfig.Port, previousGlobalPort)

	//Test case e: provide a non free port
	port, err := networking.FindFreePort()
	require.Nil(t, err, "Unexpected error in free port search")
	listeningAddress := ":" + fmt.Sprint(port)
	listener, err := net.Listen("tcp", listeningAddress)
	assert.Nil(t, err, "FindFreePort returned a non free port")
	defer listener.Close()

	usedPortProvided := portCmdLineArg{port: int(port)}
	previousGlobalPort = config.GlobalConfig.Port
	outcome = usedPortProvided.handle()
	assert.NotNil(t, outcome, "Hanlding of used port did not result in an error")
	assert.Equal(t, outcome.msg, fmt.Sprintf("Cannot listen to given port: %d", port))
	assert.Equal(t, outcome.code, CantListenToPortError)
	assert.Equal(t, config.GlobalConfig.Port, previousGlobalPort)
}

func TestHandleCommandLineArgValues(t *testing.T){
	//Lets mock the cmd line args by adding one that always succeeds
	var cmdArgs []CmdLineArg
	cmdArgs = append(cmdArgs, &successCmdLineArgMock{})
	outcome := handleCommandLineArgValues(cmdArgs)
	assert.Nil(t, outcome, "Hanlding of CmdLineArgs list should succeed")

	//Now lets add one that always fails and evaluate again
	cmdArgs = append(cmdArgs, &failedCmdLineArgMock{})
	outcome = handleCommandLineArgValues(cmdArgs)
	assert.NotNil(t, outcome, "Hanlding of CmdLineArgs list should fail")
	assert.Equal(t, outcome.msg, "error")
	assert.Equal(t, outcome.code, NormalExit)
}

type successCmdLineArgMock struct {}
func (arg *successCmdLineArgMock) register() {}
func (arg *successCmdLineArgMock) handle() *CmdLineArgError { return nil }

type failedCmdLineArgMock struct {}
func (arg *failedCmdLineArgMock) register() {}
func (arg *failedCmdLineArgMock) handle() *CmdLineArgError { return &CmdLineArgError{msg: "error", code: NormalExit} }