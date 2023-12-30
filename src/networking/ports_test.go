package networking

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//Test case: Test that funxtion IsPortFree answers correctly for used and unused ports
func TestIsPortFree(t *testing.T) {
	port, err := FindFreePort()
	require.Nil(t, err, "Unexpected error in free port search")
	assert.True(t, IsPortFree(port), "Port characterised as used although just retrieved as free")

	listeningAddress := ":" + fmt.Sprint(port)
	listener, err := net.Listen("tcp", listeningAddress)
	assert.Nil(t, err, "FindFreePort returned a non free port")
	require.False(t, IsPortFree(port), "Port characterized as free although listener is active")
	listener.Close()
}

//Test case: Test that functions FindFreePort returns a port that is not used
func findIndexOfLocalAddressHeader(lines []string) int {
	headerLine := lines[0]
	headerLine = strings.Replace(headerLine, "Local Address", "Local-Address", -1)
	headerLine = strings.Replace(headerLine, "Peer Address",  "Peer-Address", -1)
	columns := strings.Fields(headerLine)

	for i, col := range columns {
		if strings.Contains(col, "Local-Address:Port") {
			return i;
		}
	}
	return -1;
}

func parseLocalAddressPortsOfColumn(lines []string, column int) map[string]bool {
	uniquePorts := make(map[string]bool)

	for i := 1; i < len(lines); i += 1 {
		line := lines[i]
		columns := strings.Fields(line)

		if len(columns) > column {
			addressInfo := columns[column]
			re := regexp.MustCompile(`[^:]+$`)
			match := re.FindStringSubmatch(addressInfo)
			port := match[0]
			uniquePorts[port] = true
		}
	}

	return uniquePorts
}

func getUsedPorts() ([]int, error) {
	cmd := exec.Command("ss", "-tulna")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error running ss command: %v", err)
	}

	// Find the column index for "Local Address"
	lines := strings.SplitN(string(output), "\n", -1)
	localAddressIndex := findIndexOfLocalAddressHeader(lines)
	if localAddressIndex == -1 {
		return nil, fmt.Errorf("unable to determine the column index for Local Address")
	}

	//parse port numbers from command
	uniquePorts := parseLocalAddressPortsOfColumn(lines, localAddressIndex)

	// Convert the unique matches to a slice
	portsSlice := make([]int, 0, len(uniquePorts))
	for port := range uniquePorts {
		if portNum, err := strconv.Atoi(port); err == nil {
			portsSlice = append(portsSlice, portNum)
		}
	}

	return portsSlice, nil
}

func isUnix() bool {
	return runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "freebsd" || runtime.GOOS == "netbsd" || runtime.GOOS == "openbsd"
}

func searchPort(searchTarget Port, ports []int) bool {
	for _, portNum := range(ports){
		if portNum == int(searchTarget) {
			return true
		}
	}
	return false
}

func TestFindFreePortInLinux(t *testing.T){
	if !isUnix() {
		return
	}

	//Lets listen to a port to control that test works
	controlPort, err := FindFreePort()
	require.Nil(t, err, "Unexpected error in free port search")
	listeningAddress := ":" + fmt.Sprint(controlPort)
	listener, err := net.Listen("tcp", listeningAddress)
	require.Nil(t, err, "Failed to listen on a free port")
	defer listener.Close()

	usedPorts, err := getUsedPorts()
	require.Nil(t, err, "Failed to get all used ports")
	assert.True(t, searchPort(controlPort, usedPorts), "Unit test fails to report a non free port as used")

	freePort, err := FindFreePort()
	require.Nil(t, err, "Failed to get a free port")

	isUsed := searchPort(freePort, usedPorts)
	assert.False(t, isUsed, "FindFreePort returned a used port")
}