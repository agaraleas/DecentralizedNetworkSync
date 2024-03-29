package main

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/agaraleas/DecentralizedNetworkSync/config"
	"github.com/agaraleas/DecentralizedNetworkSync/logging"
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
	_, hasCorrectType = args[index].(*sharedDirectoryCmdLineArg)
	assert.True(t, hasCorrectType, fmt.Sprintf("Index %d has unexpected type instead of sharedDirectoryCmdLineArg", index))

	index += 1
	_, hasCorrectType = args[index].(*driverUrlCmdLineArg)
	assert.True(t, hasCorrectType, fmt.Sprintf("Index %d has unexpected type instead of driverUrlCmdLineArg", index))

	assert.Equal(t, len(args), index+1) //So as when adding a new cmd line arg, not forget to add a test
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

func TestdriverUrlCmdLineArgRegistering(t *testing.T) {
	require.Nil(t, flag.Lookup("driver-url"), "Cannot proceed in actual test of registering. --driver-url already exists")
	driverUrlArg := driverUrlCmdLineArg{}
	driverUrlArg.register()
	assert.NotNil(t, flag.Lookup("driver-url"), "Failed to register --driver-url flag")
}
func TestdriverUrlCmdLineArgHandling(t *testing.T) {
	logLevel := logging.Log.GetLevel()
	logging.Log.SetLevel(logrus.FatalLevel)
	defer logging.Log.SetLevel(logLevel)

	//Valid driver url with Ip case
	url := "ws://127.0.0.1:8080"
	driverUrlArg := driverUrlCmdLineArg{websocketServerUrl: url}
	outcome := driverUrlArg.handle()
	assert.Nil(t, outcome, "Hanlding of driver cmd line arg returned error")
	assert.Equal(t, config.GlobalConfig.DriverUrl, url)

	//Valid secure driver url with Ip case
	url = "wss://127.0.0.1:8080"
	driverUrlArg = driverUrlCmdLineArg{websocketServerUrl: url}
	outcome = driverUrlArg.handle()
	assert.Nil(t, outcome, "Hanlding of driver cmd line arg returned error")
	assert.Equal(t, config.GlobalConfig.DriverUrl, url)

	//Valid secure driver url with hostname case
	url = "wss://localhost:8080"
	driverUrlArg = driverUrlCmdLineArg{websocketServerUrl: url}
	outcome = driverUrlArg.handle()
	assert.Nil(t, outcome, "Hanlding of driver cmd line arg returned error")
	assert.Equal(t, config.GlobalConfig.DriverUrl, url)

	//Empty host
	url = "ws://:8080"
	previousDriverUrl := config.GlobalConfig.DriverUrl
	driverUrlArg = driverUrlCmdLineArg{websocketServerUrl: url}
	outcome = driverUrlArg.handle()
	require.NotNil(t, outcome, "Expected error in handling of invalid host but got nil")
	assert.Equal(t, outcome.msg, "Error in parsing arg --driver-url. Empty host")
	assert.Equal(t, outcome.code, InvalidDriverUrlError)
	assert.Equal(t, config.GlobalConfig.DriverUrl, previousDriverUrl)

	//Invalid hostname
	url = "ws://invalidhost:8080"
	previousDriverUrl = config.GlobalConfig.DriverUrl
	driverUrlArg = driverUrlCmdLineArg{websocketServerUrl: url}
	outcome = driverUrlArg.handle()
	require.NotNil(t, outcome, "Expected error in handling of invalid host but got nil")
	assert.Equal(t, outcome.msg, "Error in parsing arg --driver-url. Invalid host: invalidhost")
	assert.Equal(t, outcome.code, InvalidDriverUrlError)
	assert.Equal(t, config.GlobalConfig.DriverUrl, previousDriverUrl)

	//Invalid host Ip
	url = "ws://285.0.0.1:8080"
	previousDriverUrl = config.GlobalConfig.DriverUrl
	driverUrlArg = driverUrlCmdLineArg{websocketServerUrl: url}
	outcome = driverUrlArg.handle()
	require.NotNil(t, outcome, "Expected error in handling of invalid host but got nil")
	assert.Equal(t, outcome.msg, "Error in parsing arg --driver-url. Invalid host: 285.0.0.1")
	assert.Equal(t, outcome.code, InvalidDriverUrlError)
	assert.Equal(t, config.GlobalConfig.DriverUrl, previousDriverUrl)

	//Empty url
	url = ""
	previousDriverUrl = config.GlobalConfig.DriverUrl
	driverUrlArg = driverUrlCmdLineArg{websocketServerUrl: url}
	outcome = driverUrlArg.handle()
	assert.Nil(t, outcome, "Hanlding of driver cmd line arg returned error")
	assert.Equal(t, config.GlobalConfig.DriverUrl, previousDriverUrl)

	//Negative port
	url = "ws://127.0.0.1:-1"
	previousDriverUrl = config.GlobalConfig.DriverUrl
	driverUrlArg = driverUrlCmdLineArg{websocketServerUrl: url}
	outcome = driverUrlArg.handle()
	require.NotNil(t, outcome, "Expected error in handling of invalid port but got nil")
	assert.Equal(t, outcome.msg, "Error in parsing arg --driver-url. Invalid port: -1")
	assert.Equal(t, outcome.code, InvalidDriverUrlError)
	assert.Equal(t, config.GlobalConfig.DriverUrl, previousDriverUrl)

	//High port
	url = "ws://127.0.0.1:65536"
	previousDriverUrl = config.GlobalConfig.DriverUrl
	driverUrlArg = driverUrlCmdLineArg{websocketServerUrl: url}
	outcome = driverUrlArg.handle()
	require.NotNil(t, outcome, "Expected error in handling of invalid port but got nil")
	assert.Equal(t, outcome.msg, "Error in parsing arg --driver-url. Invalid port: 65536")
	assert.Equal(t, outcome.code, InvalidDriverUrlError)
	assert.Equal(t, config.GlobalConfig.DriverUrl, previousDriverUrl)

	//Non integer port
	url = "ws://127.0.0.1:a123"
	previousDriverUrl = config.GlobalConfig.DriverUrl
	driverUrlArg = driverUrlCmdLineArg{websocketServerUrl: url}
	outcome = driverUrlArg.handle()
	require.NotNil(t, outcome, "Expected error in handling of invalid port but got nil")
	assert.Equal(t, outcome.msg, "Error in parsing arg --driver-url. Invalid port: a123")
	assert.Equal(t, outcome.code, InvalidDriverUrlError)
	assert.Equal(t, config.GlobalConfig.DriverUrl, previousDriverUrl)
}

func TestHandleCommandLineArgValues(t *testing.T) {
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

type successCmdLineArgMock struct{}

func (arg *successCmdLineArgMock) register()                {}
func (arg *successCmdLineArgMock) handle() *CmdLineArgError { return nil }

type failedCmdLineArgMock struct{}

func (arg *failedCmdLineArgMock) register() {}
func (arg *failedCmdLineArgMock) handle() *CmdLineArgError {
	return &CmdLineArgError{msg: "error", code: NormalExit}
}

func TestSharedDirectoryCmdLineArgHandling(t *testing.T) {
	logLevel := logging.Log.GetLevel()
	logging.Log.SetLevel(logrus.FatalLevel)
	defer logging.Log.SetLevel(logLevel)

	// Empty path
	sharedDirectoryArg := sharedDirectoryCmdLineArg{sharedDirectoryPath: ""}
	outcome := sharedDirectoryArg.handle()
	previousGlobal := config.GlobalConfig.SharedDirectory
	assert.NotNil(t, outcome, "Hanlding of empty path did not result in an error")
	assert.Equal(t, outcome.msg, "Shared directory path cannot be empty")
	assert.Equal(t, outcome.code, InvalidSharedDirectoryError)
	assert.Equal(t, config.GlobalConfig.SharedDirectory, previousGlobal)

	// Invalid path
	sharedDirectoryArg = sharedDirectoryCmdLineArg{sharedDirectoryPath: "//invalid/path"}
	outcome = sharedDirectoryArg.handle()
	previousGlobal = config.GlobalConfig.SharedDirectory
	assert.NotNil(t, outcome, "Hanlding of invalid path did not result in an error")
	assert.Equal(t, outcome.code, InvalidSharedDirectoryError)
	assert.Equal(t, config.GlobalConfig.SharedDirectory, previousGlobal)

	// Non directory path
	tempFile, err := os.CreateTemp("", "temp_file")
	require.Nil(t, err, "Failed to create temporary file")
	defer os.Remove(tempFile.Name())
	sharedDirectoryArg = sharedDirectoryCmdLineArg{sharedDirectoryPath: tempFile.Name()}
	outcome = sharedDirectoryArg.handle()
	previousGlobal = config.GlobalConfig.SharedDirectory
	assert.NotNil(t, outcome, "Hanlding of path not pointing to directory did not result in an error")
	assert.Equal(t, outcome.msg, fmt.Sprintf("Shared directory path does not point to a directory: %s", tempFile.Name()))
	assert.Equal(t, outcome.code, InvalidSharedDirectoryError)
	assert.Equal(t, config.GlobalConfig.SharedDirectory, previousGlobal)

	// Valid directory path
	tempDirectoryPath := os.TempDir()
	sharedDirectoryArg = sharedDirectoryCmdLineArg{sharedDirectoryPath: tempDirectoryPath}
	outcome = sharedDirectoryArg.handle()
	assert.Nil(t, outcome, "Hanlding of shared directory cmd line arg returned error")
	assert.Equal(t, config.GlobalConfig.SharedDirectory, tempDirectoryPath)
}
