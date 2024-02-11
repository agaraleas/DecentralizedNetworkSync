package driver

import (
	"github.com/agaraleas/DecentralizedNetworkSync/networking"
)

type DriverWebsocketMsgHandler struct{}

func (h *DriverWebsocketMsgHandler) Handle(ws *networking.WebsocketClient, msg networking.WebsocketMsg) error {
	// Your handling logic here
	return nil
}

func (h *DriverWebsocketMsgHandler) Disconnecting(ws *networking.WebsocketClient) {
	// Handle disconnection logic here
}

type DriverCommunicator struct {
	websocketClient networking.AbstractWebsocketClient
}

func CreateDriverCommunicator() *DriverCommunicator {

	msgHandler := DriverWebsocketMsgHandler{}
	payloadRegistry := networking.WebSocketPayloadRegistry{}

	driverCommunicator := &DriverCommunicator{
		websocketClient: networking.CreateWebsocketClient(&msgHandler, &payloadRegistry),
	}
	return driverCommunicator
}

func (c *DriverCommunicator) Connect(driverUrl string) error {
	return c.websocketClient.Connect(driverUrl)
}

func (c *DriverCommunicator) Disconnect() {
	c.websocketClient.Disconnect()
}
