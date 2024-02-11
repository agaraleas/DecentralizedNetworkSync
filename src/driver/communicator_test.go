package driver

import (
	"testing"

	"github.com/agaraleas/DecentralizedNetworkSync/networking"
	"github.com/stretchr/testify/assert"
)

type MockWebsocketClient struct {
	networking.WebsocketClient // Embed the WebsocketClient interface
	connected                  bool
}

func (m *MockWebsocketClient) Connect(url string) error {
	m.connected = true
	return nil
}

func (m *MockWebsocketClient) Disconnect() {
	m.connected = false
}

func (m *MockWebsocketClient) Send(payload networking.WebSocketMsgPayload) {

}

func (m *MockWebsocketClient) Reply(payload networking.WebSocketMsgPayload, rx networking.WebsocketMsg) {

}

func TestCreateDriverCommunicator(t *testing.T) {
	communicator := CreateDriverCommunicator()

	assert.NotNil(t, communicator)
	assert.NotNil(t, communicator.websocketClient)
}

func TestDriverCommunicator_Connect(t *testing.T) {
	mockClient := &MockWebsocketClient{}
	communicator := &DriverCommunicator{
		websocketClient: mockClient,
	}

	err := communicator.Connect("test-url")

	assert.NoError(t, err)
	assert.True(t, mockClient.connected)
}

func TestDriverCommunicator_Disconnect(t *testing.T) {
	mockClient := &MockWebsocketClient{connected: true}
	communicator := &DriverCommunicator{
		websocketClient: mockClient,
	}

	communicator.Disconnect()

	assert.False(t, mockClient.connected)
}
