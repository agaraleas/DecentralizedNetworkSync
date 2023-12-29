package networking

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//Test case: Check that the obtained local IP is one of the network interfaces
func isInterfaceContained(searchTarget net.IP, networkInterfaces []net.Addr) bool {
	for _, interfaceAddress := range networkInterfaces {
		if networkIp, ok := interfaceAddress.(*net.IPNet); ok {
			if networkIp.IP.Equal(searchTarget) {
				return true
			}
		}
	}

	return false
}

func TestLocalIpRetrievalFromInterfaces(t *testing.T) {
	interfaces, err := net.InterfaceAddrs()
	require.Equal(t, err, nil, "Could not retrieve interfaces")

	localIp := GetLocalIP()
	assert.Equal(t, isInterfaceContained(localIp, interfaces), true, "local IP is missing from available interfaces")
}

//Test case: Test retriving of the local IP from a dummy UDP connection
//We will mock the dialing service to be able to control its response

//brief: mockConn
//struct which mocks a dialing connection, to be retured from the mockDialer
type mockConn struct{}

func (mc *mockConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (mc *mockConn) Write(b []byte) (n int, err error) {
	return 0, nil
}

func (mc *mockConn) Close() error {
	return nil
}

func (mc *mockConn) LocalAddr() net.Addr {
	// Return a mock local address
	return &net.UDPAddr{IP: net.IPv4(192, 168, 1, 10), Port: 12345}
}

func (mc *mockConn) RemoteAddr() net.Addr {
	return nil
}

func (mc *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (mc *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (mc *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type mockDialer struct{}

func (md *mockDialer) Dial(network, address string) (net.Conn, error) {
	return &mockConn{}, nil
}
func TestGetLocalIpThroughDialing(t *testing.T) {
	mockDialer := &mockDialer{}
	localIP, err := getLocalIpThroughDialing(mockDialer)
	require.Equal(t, err, nil, "Error when dialing")
	assert.Equal(t, net.IPv4(192, 168, 1, 10), localIP, "local IP obtained through dialing is not correct")
}