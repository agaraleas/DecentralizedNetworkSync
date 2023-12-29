package networking

import (
	"net"

	"github.com/stretchr/testify/mock"
)

type MockDialer struct {
	mock.Mock
}

func (m *MockDialer) Dial(network, address string) (net.Conn, error) {
	args := m.Called(network, address)
	return args.Get(0).(net.Conn), args.Error(1)
}
// func TestGetLocalIp(t *testing.T) {
// 	mockDialer := new(MockDialer)
// 	fakeConn, _ := net.Pipe()

// 	// Set up expectations for the Dial method
// 	mockDialer.On("Dial", "udp", "8.8.8.8:80").Return(fakeConn, nil)

// 	// Inject the mock dialer into the function
// 	oldDialer := dialer
// 	defer func() { dialer = oldDialer }()
// 	dialer = mockDialer

// 	// Call the function being tested
// 	localIP := GetLocalIP()

// 	// Assert the expected behavior
// 	assert.Equal(t, net.IPv4(127, 0, 0, 1), localIP)

// 	// Assert that the Dial method was called as expected
// 	mockDialer.AssertExpectations(t)
// }