package main

import (
	"flag"
	"fmt"
	"net"
	"testing"

	"github.com/agaraleas/DecentralizedNetworkSync/config"
	"github.com/agaraleas/DecentralizedNetworkSync/logging"
	"github.com/agaraleas/DecentralizedNetworkSync/networking"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatherCommandLineArgTemplates(t *testing.T) {
	args := gatherCommandLineArgTemplates()
	
	index := 0
	_, hasCorrectType := args[index].(*helpCmdLineArg)
	assert.True(t, hasCorrectType, fmt.Sprintf("Index %d has unexpected type instead of helpCmdLineArg", index))
	
	index += 1
	_, hasCorrectType = args[index].(*driverCmdLineArg)
	assert.True(t, hasCorrectType, fmt.Sprintf("Index %d has unexpected type instead of driverCmdLineArg", index))
	
	index += 1
	_, hasCorrectType = args[index].(*listenCmdLineArg)
	assert.True(t, hasCorrectType, fmt.Sprintf("Index %d has unexpected type instead of listenCmdLineArg", index))

	assert.Equal(t, len(args), index + 1) //So as when adding a new cmd line arg, not forget to add a test
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

func TestDriverCmdLineArgRegistering(t *testing.T) {
	require.Nil(t, flag.Lookup("driver-host"), "Cannot proceed in actual test of registering. --driver-host already exists")
	require.Nil(t, flag.Lookup("driver-port"), "Cannot proceed in actual test of registering. --driver-port already exists")
	require.Nil(t, flag.Lookup("driver-ticket"), "Cannot proceed in actual test of registering. --driver-ticket already exists")
	driverArg := driverCmdLineArg{}
	driverArg.register()
	assert.NotNil(t, flag.Lookup("driver-host"), "Failed to register --driver-host flag")
	assert.NotNil(t, flag.Lookup("driver-port"), "Failed to register --driver-port flag")
	assert.NotNil(t, flag.Lookup("driver-ticket"), "Failed to register --driver-ticket flag")
}
func TestDriverCmdLineArgHandling(t *testing.T) {
	logLevel := logging.Log.GetLevel()
	logging.Log.SetLevel(logrus.FatalLevel)
	defer logging.Log.SetLevel(logLevel)

	//Valid driver case
	driverArg := driverCmdLineArg{host: "127.0.0.1", port: 12345, ticket: "abcd"}
	outcome := driverArg.handle()
	assert.Nil(t, outcome, "Hanlding of driver cmd line arg returned error")
	assert.Equal(t, config.GlobalConfig.DriverInfo.Address.Host, net.IPv4(127, 0, 0, 1))
	assert.Equal(t, config.GlobalConfig.DriverInfo.Address.Port, networking.Port(12345))
	assert.Equal(t, config.GlobalConfig.DriverInfo.Ticket, "abcd")

	//Invalid host
	previousDriver := config.GlobalConfig.DriverInfo
	driverArg = driverCmdLineArg{host: "285.0.0.1", port: 23456, ticket: "qwerty"}
	outcome = driverArg.handle()
	require.NotNil(t, outcome, "Expected error in handling of invalid host but got nil")
	assert.Equal(t, outcome.msg, "Error in parsing arg --driver-host. Invalid host: 285.0.0.1")
	assert.Equal(t, outcome.code, InvalidHostError)
	assert.Equal(t, config.GlobalConfig.DriverInfo, previousDriver)

	//Empty host
	previousDriver = config.GlobalConfig.DriverInfo
	driverArg = driverCmdLineArg{port: 23456, ticket: "qwerty"}
	outcome = driverArg.handle()
	require.NotNil(t, outcome, "Expected error in handling of invalid host but got nil")
	assert.Equal(t, outcome.msg, "Error in parsing arg --driver-host. Invalid host: ")
	assert.Equal(t, outcome.code, InvalidHostError)
	assert.Equal(t, config.GlobalConfig.DriverInfo, previousDriver)

	//Negative port
	previousDriver = config.GlobalConfig.DriverInfo
	driverArg = driverCmdLineArg{host: "127.0.0.1", port: -1, ticket: "qwerty"}
	outcome = driverArg.handle()
	require.NotNil(t, outcome, "Expected error in handling of invalid port but got nil")
	assert.Equal(t, outcome.msg, "Error in parsing arg --driver-port. Invalid port: -1")
	assert.Equal(t, outcome.code, InvalidPortError)
	assert.Equal(t, config.GlobalConfig.DriverInfo, previousDriver)

	//High port
	previousDriver = config.GlobalConfig.DriverInfo
	driverArg = driverCmdLineArg{host: "127.0.0.1", port: 123456, ticket: "qwerty"}
	outcome = driverArg.handle()
	require.NotNil(t, outcome, "Expected error in handling of invalid port but got nil")
	assert.Equal(t, outcome.msg, "Error in parsing arg --driver-port. Invalid port: 123456")
	assert.Equal(t, outcome.code, InvalidPortError)
	assert.Equal(t, config.GlobalConfig.DriverInfo, previousDriver)

	//Empty ticket - allowed
	driverArg = driverCmdLineArg{host: "192.168.1.200", port: 55555}
	outcome = driverArg.handle()
	assert.Nil(t, outcome, "Error in parsing arg --driver-ticket. Hanlding of driver cmd line arg returned error")
	assert.Equal(t, config.GlobalConfig.DriverInfo.Address.Host, net.IPv4(192, 168, 1, 200))
	assert.Equal(t, config.GlobalConfig.DriverInfo.Ticket, "")
	assert.Equal(t, config.GlobalConfig.DriverInfo.Address.Port, networking.Port(55555))
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

func TestListenCmdLineArgHandling(t *testing.T) {
	logLevel := logging.Log.GetLevel()
	logging.Log.SetLevel(logrus.FatalLevel)
	defer logging.Log.SetLevel(logLevel)

	//Get a free port
	freePort, err := networking.FindFreePort()
	require.Nil(t, err, "Failed to get a free port to continue with actual test")

	//provide a valid listening address
	listenArg := listenCmdLineArg{address: fmt.Sprintf("192.168.101.1:%d", freePort)}
	outcome := listenArg.handle()
	assert.Nil(t, outcome, "Hanlding of listen cmd line arg returned error")
	assert.Equal(t, config.GlobalConfig.ListenAddress.Host, net.IPv4(192, 168, 101, 1))
	assert.Equal(t, config.GlobalConfig.ListenAddress.Port, freePort)

	//provide a short localhost address
	listenArg = listenCmdLineArg{address: fmt.Sprintf(":%d", freePort)}
	outcome = listenArg.handle()
	assert.Nil(t, outcome, "Hanlding of listen cmd line arg returned error")
	assert.Equal(t, config.GlobalConfig.ListenAddress.Host, net.IPv4(127, 0, 0, 1))
	assert.Equal(t, config.GlobalConfig.ListenAddress.Port, freePort)

	//dont provide a port
	previousGlobal := config.GlobalConfig.ListenAddress
	listenArg = listenCmdLineArg{address: "127.0.0.1:"}
	outcome = listenArg.handle()
	assert.NotNil(t, outcome, "Hanlding of listen cmd line arg successful while port was not given")
	assert.Equal(t, outcome.msg, "Error in parsing arg --listen. Invalid port number")
	assert.Equal(t, outcome.code, CantListenToPortError)
	assert.Equal(t, previousGlobal, config.GlobalConfig.ListenAddress)

	//provide a negative port number
	previousGlobal = config.GlobalConfig.ListenAddress
	listenArg = listenCmdLineArg{address: "127.0.0.1:-1"}
	outcome = listenArg.handle()
	assert.NotNil(t, outcome, "Hanlding of invalid port did not result in an error")
	assert.Equal(t, outcome.msg, "Error in parsing arg --listen. Invalid port number")
	assert.Equal(t, outcome.code, CantListenToPortError)
	assert.Equal(t, previousGlobal, config.GlobalConfig.ListenAddress)

	//provide a higher port number than allowed
	previousGlobal = config.GlobalConfig.ListenAddress
	listenArg = listenCmdLineArg{address: "127.0.0.1:123456"}
	outcome = listenArg.handle()
	assert.NotNil(t, outcome, "Hanlding of high port did not result in an error")
	assert.Equal(t, outcome.msg, "Error in parsing arg --listen. Invalid port number")
	assert.Equal(t, outcome.code, CantListenToPortError)
	assert.Equal(t, previousGlobal, config.GlobalConfig.ListenAddress)

	//provide a non free port
	port, err := networking.FindFreePort()
	require.Nil(t, err, "Unexpected error in free port search")
	listeningAddress := ":" + fmt.Sprint(port)
	listener, err := net.Listen("tcp", listeningAddress)
	assert.Nil(t, err, "FindFreePort returned a non free port")
	defer listener.Close()
	
	previousGlobal = config.GlobalConfig.ListenAddress
	listenArg = listenCmdLineArg{address: listeningAddress}
	outcome = listenArg.handle()
	assert.NotNil(t, outcome, "Hanlding of used port did not result in an error")
	assert.Equal(t, outcome.msg, "Error in parsing arg --listen. Port is not free")
	assert.Equal(t, outcome.code, CantListenToPortError)
	assert.Equal(t, previousGlobal, config.GlobalConfig.ListenAddress)

	//provide an empty listening address
	previousGlobal = config.GlobalConfig.ListenAddress
	listenArg = listenCmdLineArg{}
	outcome = listenArg.handle()
	assert.NotNil(t, outcome, "Hanlding of empty listen address did not result in an error")
	assert.Equal(t, outcome.msg, "Error in arg --listen. Listen address not provided")
	assert.Equal(t, outcome.code, CantListenToPortError)
	assert.Equal(t, previousGlobal, config.GlobalConfig.ListenAddress)
}