package networking

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agaraleas/DecentralizedNetworkSync/logging"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type websocketTestServer struct {
	testServer *httptest.Server
	messageChannel chan WebsocketMsg
	replyChannel chan bool
	funcChannel chan func(*websocket.Conn)
	doneChannel chan struct{}
}

func (s *websocketTestServer) setup() {
	s.messageChannel = make(chan WebsocketMsg, 10)
	s.replyChannel = make(chan bool, 10)
	s.funcChannel = make(chan func(*websocket.Conn), 10)
	s.doneChannel = make(chan struct{})
	s.testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, _ := upgrader.Upgrade(w, r, nil)
		defer conn.Close()

		block := true
		for block {
			select {
			case msg := <-s.messageChannel:
				json, _ := json.Marshal(msg)
				conn.WriteMessage(websocket.TextMessage, []byte(string(json)))
				s.replyChannel <- true
			case fun := <-s.funcChannel:
				fun(conn)
				s.replyChannel <- true
			case <-s.doneChannel:
				block = false
				break
			}
		}
	}))
}

func (s *websocketTestServer) teardown() {
	close(s.messageChannel)
	close(s.replyChannel)
	close(s.funcChannel)
	close(s.doneChannel)
	s.testServer.Close()
}

func wrapPayloadInMsg(payload WebSocketMsgPayload) WebsocketMsg {
	return WebsocketMsg {TransactionId: uuid.New(), 
		MsgType: payload.Type(),
		Payload: payload,
	}
}

func TestWebSocketPayloadRegistry(t *testing.T) {
	t.Run("Successful Registration", func(t *testing.T) {
		//Test creation
		registry := WebSocketPayloadRegistry{}
		assert.Nil(t, registry.payloadTemplates)

		//Test Register
		registry.Register(new(wsGoodByeMsg))
		assert.NotNil(t, registry.payloadTemplates)
		assert.Equal(t, len(registry.payloadTemplates), 1)
		for key, value := range registry.payloadTemplates {
			assert.Equal(t, key, "networking/wsGoodByeMsg")
			assert.Equal(t, &wsGoodByeMsg{}, *value)
		}
	})

	t.Run("Successful fetch", func(t *testing.T) {
		//Test fetch when empty
		registry := WebSocketPayloadRegistry{}
		template, found := registry.getTemplate("")
		require.False(t, found)
		require.Equal(t, nil, template)

		//Add a value
		payload := new(wsGoodByeMsg)
		registry.Register(payload)
		require.NotNil(t, registry.payloadTemplates)
		
		//Test a successful fetch
		template, found = registry.getTemplate(payload.Type())
		require.True(t, found)
		assert.Equal(t, &wsGoodByeMsg{}, template)
	})

	t.Run("Failing fetch", func(t *testing.T) {
		//Test fetch when empty
		registry := WebSocketPayloadRegistry{}
		template, found := registry.getTemplate("")
		require.False(t, found)
		require.Equal(t, nil, template)

		//Add a value
		registry.Register(new(wsGoodByeMsg))
		require.NotNil(t, registry.payloadTemplates)
		
		//Test a failing fetch
		template, found = registry.getTemplate("12345")
		require.False(t, found)
		require.Equal(t, nil, template)
	})
}

//Test wsMsgHandlingRouter
type simpleTestMsgHandler struct {
	disconnectRequested bool
	handledMsg []WebsocketMsg
}
func (h *simpleTestMsgHandler) Handle(ws *WebsocketClient, msg WebsocketMsg) error {
	h.handledMsg = append(h.handledMsg, msg)
	return nil
}
func (h *simpleTestMsgHandler) Disconnecting(ws *WebsocketClient) {
	h.disconnectRequested = true
}

type WsTestPayload struct {
	TestField string `json:"testField"`
}

func (p *WsTestPayload) Type() string {
	return "WsTestPayload"
}

func (p *WsTestPayload) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		TestField string `json:"testField"`
	}{
		TestField: p.TestField,
	})
}

func (p *WsTestPayload) UnmarshalJSON(data []byte) error {
	var temp struct {
		TestField string `json:"testField"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	p.TestField = temp.TestField
	return nil
}

func TestWsMsgHandlingRouter_Dispatch(t *testing.T) {
	t.Run("Successful Dispatching", func(t *testing.T) {
		handler := simpleTestMsgHandler{}
	
		payloadRegistry := WebSocketPayloadRegistry{}
		payloadRegistry.Register(&WsTestPayload{})
		
		router := createMsgHandlingRouterForWsClient(&handler, &payloadRegistry)
		
		tstPayload := WsTestPayload{TestField: "TestVal"}
		tstMsg := WebsocketMsg{MsgType: "WsTestPayload", Payload: &tstPayload}
		rawMsg, err := json.Marshal(tstMsg)
		require.Nil(t, err)

		wsClient := &WebsocketClient{trafficController: wsUpTrafficController{}}
		router.dispatch(rawMsg, "WsTestPayload", wsClient)

		require.Equal(t, 1, len(handler.handledMsg))
		assert.Equal(t, tstMsg, handler.handledMsg[0])
	})

	t.Run("Not allowed dispatching", func(t *testing.T) {
		handler := simpleTestMsgHandler{}
	
		payloadRegistry := WebSocketPayloadRegistry{}
		payloadRegistry.Register(&WsTestPayload{})
		
		router := createMsgHandlingRouterForWsClient(&handler, &payloadRegistry)
		
		tstPayload := WsTestPayload{TestField: "TestVal"}
		tstMsg := WebsocketMsg{MsgType: "WsTestPayload", Payload: &tstPayload}
		rawMsg, err := json.Marshal(tstMsg)
		require.Nil(t, err)

		mockLogger := new(logging.MockLogger)
		mockLogger.On("Warnf",  "Will not handle received msg '%s' due to Traffic controller. %s", mock.Anything).Once()

		wsClient := &WebsocketClient{trafficController: wsDisconnectingTrafficController{}, logger: mockLogger}
		router.dispatch(rawMsg, "WsTestPayload", wsClient)

		require.Equal(t, 0, len(handler.handledMsg))
	})

	t.Run("UnmarshallError", func(t *testing.T) {
		mockLogger := new(logging.MockLogger)
		wsClient := &WebsocketClient{logger: mockLogger}

		handler := simpleTestMsgHandler{}
		payloadRegistry := WebSocketPayloadRegistry{}
		payloadRegistry.Register(&WsTestPayload{})
		router := createMsgHandlingRouterForWsClient(&handler, &payloadRegistry)

		mockLogger.On("Errorf", "Failed to unmarshall websocket message %s", mock.Anything).Once()

		rawMsg := []byte("---------")
		err := router.dispatch(rawMsg, "WsTestPayload", wsClient)
		assert.NotNil(t, err)
		assert.Equal(t, "Failed to unmarshall websocket message" ,fmt.Sprint(err))
		mockLogger.AssertExpectations(t)
	})

	t.Run("Unknown payload type error", func(t *testing.T) {
		mockLogger := new(logging.MockLogger)
		wsClient := &WebsocketClient{logger: mockLogger}
	
		handler := simpleTestMsgHandler{}
		payloadRegistry := WebSocketPayloadRegistry{}
		payloadRegistry.Register(&WsTestPayload{})
		router := createMsgHandlingRouterForWsClient(&handler, &payloadRegistry)
	
		mockLogger.On("Warnf", "Failed to handle ws message. Type '%s' unknown to all payload registries", mock.Anything).Once()
	
		rawMsg := []byte("{}")
		err := router.dispatch(rawMsg, "Unknown", wsClient)
		assert.NotNil(t, err)
		assert.Equal(t, "Failed to handle ws message. Type 'Unknown' unknown to all payload registries" ,fmt.Sprint(err))
		mockLogger.AssertExpectations(t)
	})
}

//test CreateWebsocketClient
type dummyMessageHandler struct {}
func (h *dummyMessageHandler) Handle (ws *WebsocketClient, msg WebsocketMsg) error {return nil}
func (h *dummyMessageHandler) Disconnecting (ws *WebsocketClient) {}

func TestCreateMsgHandlingRouterForWsClient(t *testing.T){
	var dummyMsgHandler dummyMessageHandler
	var registry WebSocketPayloadRegistry
	msgHandleRouter := createMsgHandlingRouterForWsClient(&dummyMsgHandler, &registry)

	assert.Equal(t, maxMsgHandlerIndex + 1, len(msgHandleRouter.msgHandlers), "A new handler type was created without updating the max value")
	assert.Equal(t, maxMsgHandlerIndex + 1, len(msgHandleRouter.payloadRegistries), "A new handler type was created without updating the max value")
	assert.Equal(t, maxMsgHandlerIndex, 1, "A new handler type was created without updating the max value")

	assert.Equal(t, &dummyMsgHandler, msgHandleRouter.msgHandlers[customMsgHandlerIndex])
	assert.Equal(t, registry, *msgHandleRouter.payloadRegistries[customMsgHandlerIndex])
	require.IsType(t, &wsInternalMsgHandler{},  msgHandleRouter.msgHandlers[internalMsgHandlerIndex])
	internalMsgHandler := msgHandleRouter.msgHandlers[internalMsgHandlerIndex].(*wsInternalMsgHandler)
	assert.NotNil(t, internalMsgHandler.goodbyeMsgAckWaiter)

	internalPayloadRegistry := msgHandleRouter.payloadRegistries[internalMsgHandlerIndex]
	assert.Equal(t, 2, len(internalPayloadRegistry.payloadTemplates), "Added a new internal type without updating unit test")
	goodbyePayload := wsGoodByeMsg{}
	_, found := internalPayloadRegistry.getTemplate(goodbyePayload.Type())
	assert.True(t, found, "goodbyeMsg internal payload missing")
	goodbyeAckPayload := wsGoodByeMsgAck{}
	_, found = internalPayloadRegistry.getTemplate(goodbyeAckPayload.Type())
	assert.True(t, found, "goodbyeMsgAck internal payload missing")
}

func TestCreateWebsocketClient(t *testing.T) {
	var dummyMsgHandler dummyMessageHandler
	var registry WebSocketPayloadRegistry
	wsClient := CreateWebsocketClient(&dummyMsgHandler, &registry)
	require.NotNil(t, wsClient)

	assert.Nil(t, wsClient.conn)
	assert.Equal(t, &dummyMsgHandler, wsClient.msgHandleRouter.msgHandlers[customMsgHandlerIndex])
	assert.Equal(t, registry, *wsClient.msgHandleRouter.payloadRegistries[customMsgHandlerIndex])
	assert.NotNil(t, wsClient.interruptChannel)
	assert.NotNil(t, wsClient.doneChannel)
	assert.NotNil(t, &wsClient.disconnectNotifyGuard)
	assert.Equal(t, websocket.DefaultDialer, wsClient.dialer, "websocketDialer concrete implementation does not holding the default websocket dialer")
	assert.Equal(t, logging.Log, wsClient.logger)
	assert.IsType(t, wsInactiveTrafficController{},  wsClient.trafficController)
}

func TestWsGoodByeMsg(t *testing.T) {
	t.Run("MarshalJSON", func(t *testing.T) {
		msg := wsGoodByeMsg{}
		result, err := msg.MarshalJSON()
		assert.NoError(t, err, "Unexpected error during MarshalJSON")
		
		expectedJSON := `{}`
		assert.JSONEq(t, expectedJSON, string(result), "Unexpected JSON result")
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		msg := wsGoodByeMsg{}
	
		jsonData := []byte(`{}`)
		err := msg.UnmarshalJSON(jsonData)
		assert.NoError(t, err, "Unexpected error during UnmarshalJSON")
		assert.Equal(t,  wsGoodByeMsg{}, msg)
	})

	t.Run("CompleteLifecycle", func(t *testing.T) {
		msg := wsGoodByeMsg{}

		marshaled, err := msg.MarshalJSON()
		assert.NoError(t, err, "Unexpected error during MarshalJSON")
	
		err = msg.UnmarshalJSON(marshaled)
		assert.NoError(t, err, "Unexpected error during UnmarshalJSON")
	})
}

func TestWsGoodByeMsgAck(t *testing.T) {
	t.Run("MarshalJSON", func(t *testing.T) {
		msg := wsGoodByeMsgAck{}
		result, err := msg.MarshalJSON()
		assert.NoError(t, err, "Unexpected error during MarshalJSON")
		
		expectedJSON := `{}`
		assert.JSONEq(t, expectedJSON, string(result), "Unexpected JSON result")
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		msg := wsGoodByeMsgAck{}
	
		jsonData := []byte(`{}`)
		err := msg.UnmarshalJSON(jsonData)
		assert.NoError(t, err, "Unexpected error during UnmarshalJSON")
		assert.Equal(t,  wsGoodByeMsgAck{}, msg)
	})

	t.Run("CompleteLifecycle", func(t *testing.T) {
		msg := wsGoodByeMsgAck{}

		marshaled, err := msg.MarshalJSON()
		assert.NoError(t, err, "Unexpected error during MarshalJSON")

		err = msg.UnmarshalJSON(marshaled)
		assert.NoError(t, err, "Unexpected error during UnmarshalJSON")
	})
}

func TestWsInactiveTrafficController(t *testing.T) {
	t.Run("Send an internal message", func(t *testing.T) {
		msg := wrapPayloadInMsg(&wsGoodByeMsgAck{})

		controller := wsInactiveTrafficController{}
		allowed, err := controller.isAllowed(msg)

		assert.False(t, allowed)
		require.NotNil(t, err)
		assert.EqualError(t, err, "Websocket connection not established")
	})

	t.Run("Send a custom message", func(t *testing.T) {
		msg := wrapPayloadInMsg(&WsTestPayload{})

		controller := wsInactiveTrafficController{}
		allowed, err := controller.isAllowed(msg)

		assert.False(t, allowed)
		require.NotNil(t, err)
		assert.EqualError(t, err, "Websocket connection not established")
	})

	t.Run("Imminent disconnection", func(t *testing.T) {
		controller := wsInactiveTrafficController{}
		assert.False(t, controller.isDisconnectionRequested())
	})
}

func TestWsUpTrafficController(t *testing.T) {
	t.Run("Send an internal message", func(t *testing.T) {
		msg := wrapPayloadInMsg(&wsGoodByeMsgAck{})

		controller := wsUpTrafficController{}
		allowed, err := controller.isAllowed(msg)

		assert.True(t, allowed)
		assert.Nil(t, err)
	})

	t.Run("Send a custom message", func(t *testing.T) {
		msg := wrapPayloadInMsg(&WsTestPayload{})

		controller := wsUpTrafficController{}
		allowed, err := controller.isAllowed(msg)

		assert.True(t, allowed)
		assert.Nil(t, err)
	})

	t.Run("Imminent disconnection", func(t *testing.T) {
		controller := wsUpTrafficController{}
		assert.False(t, controller.isDisconnectionRequested())
	})
}

func TestDisconnectionUpTrafficController(t *testing.T) {
	t.Run("Send a custom message", func(t *testing.T) {
		msg := wrapPayloadInMsg(&WsTestPayload{})

		controller := wsDisconnectingTrafficController{}
		allowed, err := controller.isAllowed(msg)

		assert.False(t, allowed)
		require.NotNil(t, err)
		assert.EqualError(t, err, "Msg type 'WsTestPayload' is disregarded during disconnection")
	})

	t.Run("Send a Goodbye message", func(t *testing.T) {
		msg := wrapPayloadInMsg(&wsGoodByeMsg{})

		controller := wsDisconnectingTrafficController{}
		allowed, err := controller.isAllowed(msg)

		assert.True(t, allowed)
		assert.Nil(t, err)
	})

	t.Run("Send a Goodbye ack", func(t *testing.T) {
		msg := wrapPayloadInMsg(&wsGoodByeMsgAck{})

		controller := wsDisconnectingTrafficController{}
		allowed, err := controller.isAllowed(msg)

		assert.True(t, allowed)
		assert.Nil(t, err)
	})

	t.Run("Imminent disconnection", func(t *testing.T) {
		controller := wsDisconnectingTrafficController{}
		assert.True(t, controller.isDisconnectionRequested())
	})
}

//Test WebSocketClient::Connect
type MockDialer struct {
	mock.Mock
}

func (m *MockDialer) Dial(urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error) {
	args := m.Called(urlStr, requestHeader)

	var mockConn *websocket.Conn
	if args.Get(0) != nil {
		mockConn = args.Get(0).(*websocket.Conn)
	}

	var httpRes *http.Response
	if args.Get(1) != nil {
		httpRes = args.Get(1).(*http.Response)
	}

	return mockConn, httpRes, args.Error(2)
}

func TestWebsocketClient_Connect(t *testing.T) {
	t.Run("Successful Connection", func(t *testing.T) {
		//Create a basic websocket client
		mockDialer := new(MockDialer)
		mockLogger := new(logging.MockLogger)
		ws := &WebsocketClient{dialer: mockDialer, logger: mockLogger}

		//Set up dialing expectations
		mockDialer.On("Dial", "ws://host-url:8080/ws", http.Header(nil)).Return(&websocket.Conn{}, &http.Response{}, nil)

		// Create a mock logger and set up logging expectations
		mockLogger.On("Debugf", mock.Anything, mock.Anything).Times(1)

		// Call the Connect function
		err := ws.Connect("ws://host-url:8080/ws")

		// Assert that there are no errors
		assert.NoError(t, err, "Unexpected error during WebsocketClient::Connect")
		assert.IsType(t, wsUpTrafficController{},  ws.trafficController)

		// Assert that the logger was called with the correct message
		mockLogger.AssertExpectations(t)
	})
	
	t.Run("UrlParsingFailure", func(t *testing.T) {
		//Create a basic websocket client
		mockDialer := new(MockDialer)
		mockLogger := new(logging.MockLogger)
		ws := &WebsocketClient{dialer: mockDialer, logger: mockLogger}

		//Set up the mock logger to expect the error
		mockLogger.On("Errorf", "WebsocketClient::Connect Failed to parse url %s", []interface{}{"ws://host-url:port/ws"}).Once()

		// Call the Connect function with an invalid URL
		err := ws.Connect("ws://host-url:port/ws")

		// Assert that the error is not nil
		assert.Error(t, err, "Expected error during Connect")
		assert.IsType(t, nil,  ws.trafficController)

		mockLogger.AssertExpectations(t)
	})

	t.Run("DialingFailure", func(t *testing.T) {
		//Create a basic websocket client
		mockDialer := new(MockDialer)
		mockLogger := new(logging.MockLogger)
		ws := &WebsocketClient{dialer: mockDialer, logger: mockLogger}

		// Set up expectations for URL parsing and dialing
		mockDialer.On("Dial", "ws://host-url:8080/ws", http.Header(nil)).Return(nil, nil, assert.AnError)
		mockLogger.On("Errorf", mock.Anything, mock.Anything).Once()

		// Call the Connect function
		err := ws.Connect("ws://host-url:8080/ws")

		// Assert that the error is not nil
		assert.Error(t, err, "Expected error during Connect")
		assert.IsType(t, nil,  ws.trafficController)

		// Assert that the logger was called with the correct error message
		mockLogger.AssertExpectations(t)
	})
}

func TestWebsocketClient_Disconnect(t *testing.T) {
	t.Run("Graceful disconnection", func(t *testing.T) {
		//Startup dummy server
		s := &websocketTestServer{}
		s.setup()
		defer s.teardown()

		// Create a WebSocket client and connect to the test server
		wsURL := strings.Replace(s.testServer.URL, "http", "ws", 1)
		wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer wsConn.Close()

		// Initialize the WebsocketClient with the connected WebSocket
		payloadRegistry := WebSocketPayloadRegistry{}
		payloadRegistry.Register(new(WsTestPayload))
		msgHandler := synchronousTestMsgReceiver{receivedMessages: make(chan WebsocketMsg, 1), disconnectAction: make(chan bool, 1)}
		mockLogger := new(logging.MockLogger)
		wsClient := CreateWebsocketClient(&msgHandler, &WebSocketPayloadRegistry{})
		wsClient.conn = wsConn
		wsClient.logger = mockLogger

		//Prepare mock logger for the scenario
		mockLogger.On("Debugf", "Handling websocket disconnection of conn %p", mock.Anything).Once()
		mockLogger.On("Debugf", "Sending WebSocket '%s' msg with transaction Id %s", mock.Anything).Once()
		mockLogger.On("Debugf", "WebSocket msg with transaction Id %s successfully sent", mock.Anything).Once()
		mockLogger.On("Debugf", "Received done for ws connection %p", mock.Anything).Once()
		mockLogger.On("Debugf", "Websocket connection %p closed due to imminent disconnection", mock.Anything).Once()
		mockLogger.On("Infof", "Terminating websocket connection %p", mock.Anything).Once()

		//start reading and writing of wsClient
		go wsClient.readPump()
		go wsClient.writePump()

		//Lets prepare server side. Serveer should:
		//1. Get a goodbye message
		//2. Acknowledge it
		//3. Wait for client to close the connection

		//Prepare for step one: Set remote peer listening for a goodbye msg
		receivedGoodbyeMsg := false
		_ = receivedGoodbyeMsg
		closedGracefully := false
		_ = closedGracefully
		receptionSync := make(chan bool, 1)
		_ = receptionSync
		goodbyeMsgReception := func (remoteConn *websocket.Conn) {
			//Execute step 1: Wait to receive a goodbye message
			receptionSync <- true
			wsMsgType, txt, err := remoteConn.ReadMessage()
			assert.Equal(t, websocket.TextMessage, wsMsgType)
			assert.Nil(t, err)

			expectedPayload :=  &wsGoodByeMsg{}
			expectedType := expectedPayload.Type()
			msg := WebsocketMsg { Payload: expectedPayload}
			err = json.Unmarshal(txt, &msg)
			assert.Nil(t, err)
			assert.Equal(t, expectedType, msg.MsgType)
			assert.NotEmpty(t, msg.TransactionId.String())

			var transactionId uuid.UUID
			if expectedType == msg.MsgType {
				receivedGoodbyeMsg = true
				transactionId = msg.TransactionId
			}

			// Execute step 2: mock an acknowledgment from remote peer
			ackPayload := wsGoodByeMsgAck{}
			ack := WebsocketMsg{TransactionId: transactionId, MsgType: ackPayload.Type(), Payload: &ackPayload}
			json, _ := json.Marshal(ack)
			remoteConn.WriteMessage(websocket.TextMessage, []byte(string(json)))

			//Execute step 3: Wait for client to gracefully close the connection
			_, _, err = remoteConn.ReadMessage()
			assert.NotNil(t, err)
			closedGracefully = websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway)
		}

		//Start disconnection on another goroutine to keep this one for result assertion
		s.funcChannel <- goodbyeMsgReception
		<- receptionSync
		go wsClient.Disconnect()
		<- s.replyChannel
		
		//Execute step 1: Check that client sent a goodbye msg to peer
		assert.True(t, closedGracefully)
		assert.True(t, receivedGoodbyeMsg)
		assert.IsType(t, wsDisconnectingTrafficController{},  wsClient.trafficController)

		//We cannot have a smarter way to hold processing the mallformed message
		//since no msg handling will be done
		for len(mockLogger.Calls) < 6 {
			time.Sleep(2 * time.Millisecond)
		}

		mockLogger.AssertExpectations(t)
	})

	t.Run("Server closed connection upon disconnect request", func(t *testing.T) {
		//Startup dummy server
		s := &websocketTestServer{}
		s.setup()
		defer s.teardown()

		// Create a WebSocket client and connect to the test server
		wsURL := strings.Replace(s.testServer.URL, "http", "ws", 1)
		wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer wsConn.Close()

		// Initialize the WebsocketClient with the connected WebSocket
		msgHandler := synchronousTestMsgReceiver{receivedMessages: make(chan WebsocketMsg, 1), disconnectAction: make(chan bool, 1)}
		mockLogger := new(logging.MockLogger)
		wsClient := CreateWebsocketClient(&msgHandler, &WebSocketPayloadRegistry{})
		wsClient.conn = wsConn
		wsClient.logger = mockLogger

		//Prepare mock logger for the scenario
		mockLogger.On("Debugf", "Handling websocket disconnection of conn %p", mock.Anything).Once()
		mockLogger.On("Debugf", "Sending WebSocket '%s' msg with transaction Id %s", mock.Anything).Once()
		mockLogger.On("Debugf", "WebSocket msg with transaction Id %s successfully sent", mock.Anything).Once()
		mockLogger.On("Debugf", "Received done for ws connection %p", mock.Anything).Once()
		mockLogger.On("Debugf", "Websocket connection %p closed due to imminent disconnection", mock.Anything).Once()
		mockLogger.On("Infof", "Terminating websocket connection %p", mock.Anything).Once()

		//start reading and writing of wsClient
		go wsClient.readPump()
		go wsClient.writePump()

		//Lets prepare server side. Serveer should:
		//1. Get a goodbye message
		//2. Close the connection

		//Prepare for step one: Set remote peer listening for a goodbye msg
		receivedGoodbyeMsg := false
		_ = receivedGoodbyeMsg
		receptionSync := make(chan bool, 1)
		_ = receptionSync
		goodbyeMsgReception := func (remoteConn *websocket.Conn) {
			//Execute step 1: Wait to receive a goodbye message
			receptionSync <- true
			wsMsgType, txt, err := remoteConn.ReadMessage()
			assert.Equal(t, websocket.TextMessage, wsMsgType)
			assert.Nil(t, err)

			expectedPayload :=  &wsGoodByeMsg{}
			expectedType := expectedPayload.Type()
			msg := WebsocketMsg { Payload: expectedPayload}
			err = json.Unmarshal(txt, &msg)
			assert.Nil(t, err)
			assert.Equal(t, expectedType, msg.MsgType)
			assert.NotEmpty(t, msg.TransactionId.String())

			//Execute step 2: close the connection unexpectedly
			remoteConn.Close()
		}

		//Start disconnection on another goroutine to keep this one for result assertion
		s.funcChannel <- goodbyeMsgReception
		<- receptionSync
		go wsClient.Disconnect()
		<- s.replyChannel

		//We cannot have a smarter way to hold processing the mallformed message
		//since no msg handling will be done
		for len(mockLogger.Calls) < 3 {
			time.Sleep(2 * time.Millisecond)
		}

		mockLogger.AssertExpectations(t)
	})

	t.Run("Server sent another message upon disconnect request", func(t *testing.T) {
		//Startup dummy server
		s := &websocketTestServer{}
		s.setup()
		defer s.teardown()

		// Create a WebSocket client and connect to the test server
		wsURL := strings.Replace(s.testServer.URL, "http", "ws", 1)
		wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer wsConn.Close()

		// Initialize the WebsocketClient with the connected WebSocket
		payloadRegistry := WebSocketPayloadRegistry{}
		payloadRegistry.Register(new(WsTestPayload))
		msgHandler := synchronousTestMsgReceiver{receivedMessages: make(chan WebsocketMsg, 1), disconnectAction: make(chan bool, 1)}
		mockLogger := new(logging.MockLogger)
		wsClient := CreateWebsocketClient(&msgHandler, &payloadRegistry)
		wsClient.conn = wsConn
		wsClient.logger = mockLogger

		//Prepare mock logger for the scenario
		mockLogger.On("Debugf", "Handling websocket disconnection of conn %p", mock.Anything).Once()
		mockLogger.On("Debugf", "Sending WebSocket '%s' msg with transaction Id %s", mock.Anything).Once()
		mockLogger.On("Debugf", "WebSocket msg with transaction Id %s successfully sent", mock.Anything).Once()
		mockLogger.On("Debugf", "Received done for ws connection %p", mock.Anything).Once()
		mockLogger.On("Debugf", "Websocket connection %p closed due to imminent disconnection", mock.Anything).Once()
		mockLogger.On("Infof", "Terminating websocket connection %p", mock.Anything).Once()
		mockLogger.On("Warnf", "Will not handle received msg '%s' due to Traffic controller. %s", []interface {}{"Msg type 'WsTestPayload' is disregarded during disconnection"}).Once()

		//start reading and writing of wsClient
		go wsClient.readPump()
		go wsClient.writePump()

		//Lets prepare server side. Serveer should:
		//1. Get a goodbye message
		//2. Close the connection

		//Prepare for step one: Set remote peer listening for a goodbye msg
		receivedGoodbyeMsg := false
		_ = receivedGoodbyeMsg
		var transactionId uuid.UUID
		_ = transactionId
		goodbyeMsgReception := func (remoteConn *websocket.Conn) {
			//Wait to receive a goodbye message
			wsMsgType, txt, err := remoteConn.ReadMessage()
			require.Equal(t, websocket.TextMessage, wsMsgType)
			require.Nil(t, err)

			expectedPayload :=  &wsGoodByeMsg{}
			expectedType := expectedPayload.Type()
			msg := WebsocketMsg { Payload: expectedPayload}
			err = json.Unmarshal(txt, &msg)
			assert.Nil(t, err)
			assert.Equal(t, expectedType, msg.MsgType)
			assert.NotEmpty(t, msg.TransactionId.String())

			if expectedType == msg.MsgType {
				receivedGoodbyeMsg = true
				transactionId = msg.TransactionId
			}
		}

		//Start disconnection on another goroutine to keep this one for result assertion
		go wsClient.Disconnect()
		
		//Execute step 1: Check that client sent a goodbye msg to peer
		s.funcChannel <- goodbyeMsgReception
		<- s.replyChannel
		assert.True(t, receivedGoodbyeMsg)
		assert.IsType(t, wsDisconnectingTrafficController{},  wsClient.trafficController)

		// Execute step 2: send a random message
		randomPayload := WsTestPayload{}
		randomMsg := WebsocketMsg{TransactionId: transactionId, MsgType: randomPayload.Type(), Payload: &randomPayload}
		s.messageChannel <- randomMsg
		<- s.replyChannel

		// Execute step 3: send the acknowledgment from remote peer
		ackPayload := wsGoodByeMsgAck{}
		ack := WebsocketMsg{TransactionId: transactionId, MsgType: ackPayload.Type(), Payload: &ackPayload}
		s.messageChannel <- ack
		<- s.replyChannel

		//We cannot have a smarter way to hold processing the mallformed message
		//since no msg handling will be done
		for len(mockLogger.Calls) < 6 {
			time.Sleep(2 * time.Millisecond)
		}

		mockLogger.AssertExpectations(t)
	})
}

//Test WebsocketClient::Send
func TestWebsocketClient_Send(t *testing.T) {
	sendChannel := make(chan WebsocketMsg, websocketMsgSendQueueSize)
	wsClient := &WebsocketClient{sendChannel: sendChannel}

	processEndSync := make(chan struct{})
	tstPayload := &WsTestPayload{}

	go func(){
		msg := <- wsClient.sendChannel
		assert.NotEqual(t, uuid.UUID{}, msg.TransactionId)
		assert.Equal(t, tstPayload.Type(), msg.MsgType)
		assert.Equal(t, tstPayload, msg.Payload)
		close(processEndSync)
	}()

	wsClient.Send(tstPayload)
	<-processEndSync
}

func TestWebsocketClient_Reply(t *testing.T) {
	sendChannel := make(chan WebsocketMsg, websocketMsgSendQueueSize)
	wsClient := &WebsocketClient{sendChannel: sendChannel}

	processEndSync := make(chan struct{})
	replyPayload := &WsTestPayload{}
	srcMsg := wrapPayloadInMsg(&WsTestPayload{})

	go func(){
		msg := <- wsClient.sendChannel
		assert.Equal(t, srcMsg.TransactionId, msg.TransactionId)
		assert.Equal(t, replyPayload.Type(), msg.MsgType)
		assert.Equal(t, replyPayload, msg.Payload)
		close(processEndSync)
	}()

	wsClient.Reply(replyPayload, srcMsg)
	<-processEndSync
}


//Test handleReceivedMsg
func TestWebsocketClient_handleReceivedMsg(t *testing.T) {
	t.Run("Handling a valid message", func(t *testing.T) {
		handler := simpleTestMsgHandler{}
	
		payloadRegistry := WebSocketPayloadRegistry{}
		payloadRegistry.Register(&WsTestPayload{})
		
		router := createMsgHandlingRouterForWsClient(&handler, &payloadRegistry)
		wsClient := &WebsocketClient{msgHandleRouter: router, trafficController: wsUpTrafficController{}}
		
		tstPayload := WsTestPayload{TestField: "TestVal"}
		tstMsg := WebsocketMsg{MsgType: "WsTestPayload", Payload: &tstPayload}
		rawMsg, err := json.Marshal(tstMsg)
		require.Nil(t, err)

		wsClient.handleReceivedMsg(rawMsg)
		require.Equal(t, 1, len(handler.handledMsg))
		assert.Equal(t, tstMsg, handler.handledMsg[0])
	})

	t.Run("Handling a not valid message", func(t *testing.T) {
		mockLogger := new(logging.MockLogger)
		wsClient := &WebsocketClient{logger: mockLogger}

		mockLogger.On("Errorf", "Failed to unmarshall type of websocket message %s", mock.Anything).Once()
		
		invalidMsg := []byte("-----")
		wsClient.handleReceivedMsg(invalidMsg)

		mockLogger.AssertExpectations(t)
	})
}


//Test TestWebsocketClient_ReadPump
//Websocket message cacher for test reception
type synchronousTestMsgReceiver struct {
	receivedMessages chan WebsocketMsg
	disconnectAction chan bool
}

func (c *synchronousTestMsgReceiver) Handle(ws *WebsocketClient, msg WebsocketMsg) error {
	c.receivedMessages <- msg
	return nil
}
	
func (c* synchronousTestMsgReceiver) Disconnecting(ws *WebsocketClient){
	c.disconnectAction <- true
}

func TestWebsocketClient_ReadPump(t *testing.T) {
	t.Run("SuccessfulReception", func(t *testing.T) {
		//Startup dummy server
		s := &websocketTestServer{}
		s.setup()
		defer s.teardown()

		// Create a WebSocket client and connect to the test server
		wsURL := strings.Replace(s.testServer.URL, "http", "ws", 1)
		wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer wsConn.Close()

		//Create a cacher to receive Websocket messages
		msgHandler := synchronousTestMsgReceiver{receivedMessages: make(chan WebsocketMsg, 10)}
		payloadRegistry := WebSocketPayloadRegistry{}
		payloadRegistry.Register(new(WsTestPayload))

		// Initialize the WebsocketClient with the connected WebSocket
		wsClient := CreateWebsocketClient(&msgHandler, &payloadRegistry)
		wsClient.trafficController = wsUpTrafficController{}
		wsClient.conn = wsConn

		// Test the readPump function - Successful reception of a message
		go wsClient.readPump()

		// Send a message to the WebSocket client via the test server
		tstPayload := WsTestPayload{TestField: "TestVal"}
		tstMsg := WebsocketMsg{MsgType: "WsTestPayload", Payload: &tstPayload}
		s.messageChannel <- tstMsg

		// Wait for the server to reply and the message to process
		<-s.replyChannel
		receivedMsg := <-msgHandler.receivedMessages
		assert.Equal(t, receivedMsg, tstMsg)										
	})

	t.Run("MallformedMsg", func(t *testing.T) {
		//Startup dummy server
		s := &websocketTestServer{}
		s.setup()
		defer s.teardown()

		// Create a WebSocket client and connect to the test server
		wsURL := strings.Replace(s.testServer.URL, "http", "ws", 1)
		wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer wsConn.Close()

		// Initialize the WebsocketClient with the connected WebSocket
		msgHandler := synchronousTestMsgReceiver{receivedMessages: make(chan WebsocketMsg, 10)}
		mockLogger := new(logging.MockLogger)
		wsClient := CreateWebsocketClient(&msgHandler, &WebSocketPayloadRegistry{})
		wsClient.conn = wsConn
		wsClient.logger = mockLogger
		wsClient.trafficController = wsUpTrafficController{}

		//Prepare mock logger for the scenario
		mockLogger.On("Errorf", "Failed to read Websocket Message from connection %p. Error: %v", mock.Anything).Once()
		mockLogger.On("Infof",  "Terminating websocket connection %p", mock.Anything).Once()
		mockLogger.On("Debugf", "Handling websocket disconnection of conn %p", mock.Anything).Once()

		//Run the readPump function in another go routine to test
		go wsClient.readPump()

		// Send a mallformed message to the WebSocket client via the test server
		f := func (conn *websocket.Conn) {
			malformedCloseFrame := []byte("This is a malformed close frame")
			conn.WriteMessage(websocket.CloseMessage, malformedCloseFrame)
		}
		s.funcChannel <- f

		// Wait for the server to reply
		<-s.replyChannel

		//We cannot have a smarter way to hold processing the mallformed message
		//since no msg handling will be done
		for len(mockLogger.Calls) < 2 {
			time.Sleep(2 * time.Millisecond)
		}

		//Assert that the logger was called with the correct error message
		mockLogger.AssertExpectations(t)		
	})

	t.Run("Graceful connection close", func(t *testing.T) {
		//Startup dummy server
		s := &websocketTestServer{}
		s.setup()
		defer s.teardown()

		// Create a WebSocket client and connect to the test server
		wsURL := strings.Replace(s.testServer.URL, "http", "ws", 1)
		wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer wsConn.Close()

		// Initialize the WebsocketClient with the connected WebSocket
		msgHandler := synchronousTestMsgReceiver{receivedMessages: make(chan WebsocketMsg, 1), disconnectAction: make(chan bool, 1)}
		mockLogger := new(logging.MockLogger)
		wsClient := CreateWebsocketClient(&msgHandler, &WebSocketPayloadRegistry{})
		wsClient.conn = wsConn
		wsClient.logger = mockLogger

		//Prepare mock logger for the scenario
		mockLogger.On("Debugf", "Remote peer closed ws connection %p", mock.Anything).Once()
		mockLogger.On("Debugf", "Handling websocket disconnection of conn %p", mock.Anything).Once()
		mockLogger.On("Infof", "Terminating websocket connection %p", mock.Anything).Once()

		//Run the readPump function in another go routine to test
		go wsClient.readPump()

		// Send a close message to the WebSocket client via the test server
		f := func (remoteConn *websocket.Conn) {
			closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Connection closed gracefully")
			remoteConn.WriteMessage(websocket.CloseMessage, closeMessage)
		}
		s.funcChannel <- f

		// Wait for the server to reply
		<-s.replyChannel

		//We cannot have a smarter way to hold processing the mallformed message
		//since no msg handling will be done
		for len(mockLogger.Calls) < 2 {
			time.Sleep(2 * time.Millisecond)
		}

		//Check if disconnect action was triggered
		disconnectHandled := <- msgHandler.disconnectAction
		assert.True(t, disconnectHandled)
	
		//Assert that the logger was called with the correct error message
		mockLogger.AssertExpectations(t)
	})

	t.Run("Goodbye request handling", func(t *testing.T) {
		//Startup dummy server
		s := &websocketTestServer{}
		s.setup()
		defer s.teardown()
		
		// Create a WebSocket client and connect to the test server
		wsURL := strings.Replace(s.testServer.URL, "http", "ws", 1)
		wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer wsConn.Close()

		// Initialize the WebsocketClient with the connected WebSocket
		msgHandler := synchronousTestMsgReceiver{receivedMessages: make(chan WebsocketMsg, 1), disconnectAction: make(chan bool, 1)}
		mockLogger := new(logging.MockLogger)
		wsClient := CreateWebsocketClient(&msgHandler, &WebSocketPayloadRegistry{})
		wsClient.conn = wsConn
		wsClient.logger = mockLogger
		wsClient.trafficController = wsUpTrafficController{}

		//Prepare mock logger for the scenario
		mockLogger.On("Debugf", "Handling websocket disconnection of conn %p", mock.Anything).Once()
		mockLogger.On("Debugf", "Sending WebSocket '%s' msg with transaction Id %s", mock.Anything).Once()
		mockLogger.On("Debugf", "WebSocket msg with transaction Id %s successfully sent", mock.Anything).Once()

		//Run the readPump function in another go routine to test.
		//For this test we will need the write pump as well since an acknowledgment must be sent
		go wsClient.readPump()
		go wsClient.writePump()

		//Check that server gets a reply acknowledgment
		serverAcknowledged := false
		_ = serverAcknowledged
		f := func (remoteConn *websocket.Conn) {
			//Send a goodbye meessage and 
			tstPayload := wsGoodByeMsg{}
			goodbyeMsg := WebsocketMsg{MsgType: tstPayload.Type(), Payload: &tstPayload}
			jsonMsg, _ := json.Marshal(goodbyeMsg)
			remoteConn.WriteMessage(websocket.TextMessage, []byte(string(jsonMsg)))

			//Wait for the acknowledgment
			wsMsgType, txt, err := remoteConn.ReadMessage()
			assert.Equal(t, websocket.TextMessage, wsMsgType)
			assert.Nil(t, err)

			expectedPayload :=  &wsGoodByeMsgAck{}
			expectedType := expectedPayload.Type()
			msg := WebsocketMsg { Payload: expectedPayload}
			err = json.Unmarshal(txt, &msg)
			assert.Nil(t, err)
			assert.Equal(t, expectedType, msg.MsgType)

			if expectedType == msg.MsgType {
				serverAcknowledged = true
			}
		}
		
		// Wait for the server goodbye ping - pong
		s.funcChannel <- f
		<-s.replyChannel

		//Check that disconnect action was executed and server got the acknowledgment
		disconnectHandled := <- msgHandler.disconnectAction
		assert.True(t, disconnectHandled)
		assert.True(t, serverAcknowledged)
	
		//Assert that the logger was called with the correct error message
		mockLogger.AssertExpectations(t)
		
		//Lets change the logger to avoid panics caused by unexpected log calls to mock loggeer on the teardown
		nilLogger := new(logging.NilLogger)
		wsClient.logger = nilLogger
	})
}

func TestWebsocketClient_WritePump(t *testing.T) {
	t.Run("Successful Send", func(t *testing.T) {
		//Startup dummy server
		s := &websocketTestServer{}
		s.setup()
		defer s.teardown()
		
		// Create a WebSocket client and connect to the test server
		wsURL := strings.Replace(s.testServer.URL, "http", "ws", 1)
		wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer wsConn.Close()

		// Initialize the WebsocketClient with the connected WebSocket
		msgHandler := synchronousTestMsgReceiver{receivedMessages: make(chan WebsocketMsg, 1), disconnectAction: make(chan bool, 1)}
		mockLogger := new(logging.MockLogger)
		wsClient := CreateWebsocketClient(&msgHandler, &WebSocketPayloadRegistry{})
		wsClient.conn = wsConn
		wsClient.logger = mockLogger
		wsClient.trafficController = wsUpTrafficController{}

		//Prepare mock logger for the scenario
		mockLogger.On("Debugf", "Sending WebSocket '%s' msg with transaction Id %s", mock.Anything).Once()
		mockLogger.On("Debugf", "WebSocket msg with transaction Id %s successfully sent", mock.Anything).Once()

		//start the pump on another go routine
		go wsClient.writePump()

		msg := wrapPayloadInMsg(&WsTestPayload{})
		wsClient.sendChannel <- msg

		//We cannot have a smarter way to hold processing the mallformed message
		//since no msg handling will be done
		for len(mockLogger.Calls) < 2 {
			time.Sleep(2 * time.Millisecond)
		}

		//Assert that the logger was called with the correct error message
		mockLogger.AssertExpectations(t)
		
		//Lets change the logger to avoid panics caused by unexpected log calls to mock loggeer on the teardown
		nilLogger := new(logging.NilLogger)
		wsClient.logger = nilLogger
	})

	t.Run("Sending a message when not connected", func(t *testing.T) {
		// Initialize the WebsocketClient with the connected WebSocket
		msgHandler := synchronousTestMsgReceiver{receivedMessages: make(chan WebsocketMsg, 1), disconnectAction: make(chan bool, 1)}
		mockLogger := new(logging.MockLogger)
		wsClient := CreateWebsocketClient(&msgHandler, &WebSocketPayloadRegistry{})
		wsClient.logger = mockLogger

		//Prepare mock logger for the scenario
		mockLogger.On("Debugf", "Sending WebSocket '%s' msg with transaction Id %s", mock.Anything).Once()
		mockLogger.On("Warnf", "Disregarding '%s' msg with transaction Id %s. Traffic controller replied: %s",  []interface {}{"Websocket connection not established"}).Once()

		//start the pump on another go routine
		go wsClient.writePump()

		msg := wrapPayloadInMsg(&WsTestPayload{})
		wsClient.sendChannel <- msg

		//We cannot have a smarter way to hold processing the mallformed message
		//since no msg handling will be done
		for len(mockLogger.Calls) < 2 {
			time.Sleep(2 * time.Millisecond)
		}

		//Assert that the logger was called with the correct error message
		mockLogger.AssertExpectations(t)
		
		//Lets change the logger to avoid panics caused by unexpected log calls to mock loggeer on the teardown
		nilLogger := new(logging.NilLogger)
		wsClient.logger = nilLogger
	})

	t.Run("Sending under connectivity error", func(t *testing.T) {
		//Startup dummy server
		s := &websocketTestServer{}
		s.setup()
		defer s.teardown()
		
		// Create a WebSocket client and connect to the test server
		wsURL := strings.Replace(s.testServer.URL, "http", "ws", 1)
		wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		
		//Close the connection for the test scenario
		wsConn.Close()

		// Initialize the WebsocketClient with the connected WebSocket
		msgHandler := synchronousTestMsgReceiver{receivedMessages: make(chan WebsocketMsg, 1), disconnectAction: make(chan bool, 1)}
		mockLogger := new(logging.MockLogger)
		wsClient := CreateWebsocketClient(&msgHandler, &WebSocketPayloadRegistry{})
		wsClient.conn = wsConn
		wsClient.logger = mockLogger
		wsClient.trafficController = wsUpTrafficController{}

		//Prepare mock logger for the scenario
		mockLogger.On("Debugf", "Sending WebSocket '%s' msg with transaction Id %s", mock.Anything).Once()
		mockLogger.On("Errorf", "Failed to send WebSocket msg %s through %p",  mock.Anything).Once()
		mockLogger.On("Debugf", "Stopping write pump of WebSocket client for ws connection %p", mock.Anything).Once()
		mockLogger.On("Infof", "Terminating websocket connection %p",  mock.Anything).Once()
		mockLogger.On("Debugf", "Handling websocket disconnection of conn %p", mock.Anything).Once()

		//start the pump on another go routine
		go wsClient.writePump()

		msg := wrapPayloadInMsg(&WsTestPayload{})
		wsClient.sendChannel <- msg

		//We cannot have a smarter way to hold processing the mallformed message
		//since no msg handling will be done
		for len(mockLogger.Calls) < 2 {
			time.Sleep(2 * time.Millisecond)
		}

		//Assert that the logger was called with the correct error message
		mockLogger.AssertExpectations(t)
		
		//Lets change the logger to avoid panics caused by unexpected log calls to mock loggeer on the teardown
		nilLogger := new(logging.NilLogger)
		wsClient.logger = nilLogger
	})

	t.Run("Test done", func(t *testing.T) {
		// Initialize the WebsocketClient with the connected WebSocket
		msgHandler := synchronousTestMsgReceiver{receivedMessages: make(chan WebsocketMsg, 1), disconnectAction: make(chan bool, 1)}
		mockLogger := new(logging.MockLogger)
		wsClient := CreateWebsocketClient(&msgHandler, &WebSocketPayloadRegistry{})
		wsClient.logger = mockLogger

		//Prepare mock logger for the scenario
		mockLogger.On("Debugf", "Received done for ws connection %p", mock.Anything).Once()
		mockLogger.On("Infof", "Terminating websocket connection %p",  mock.Anything).Once()
		mockLogger.On("Debugf", "Handling websocket disconnection of conn %p", mock.Anything).Once()

		//start the pump on another go routine
		go wsClient.writePump()

		//First send a message to know that it goroutine started
		mockLogger.On("Debugf", "Sending WebSocket '%s' msg with transaction Id %s", mock.Anything).Once()
		mockLogger.On("Warnf", "Disregarding '%s' msg with transaction Id %s. Traffic controller replied: %s",  []interface {}{"Websocket connection not established"}).Once()
		msg := wrapPayloadInMsg(&WsTestPayload{})
		wsClient.sendChannel <- msg
		for len(mockLogger.Calls) < 2 {
			time.Sleep(2 * time.Millisecond)
		}

		//Send the done signal
		wsClient.doneChannel <- true

		//We cannot have a smarter way to hold processing the mallformed message
		//since no msg handling will be done
		for len(mockLogger.Calls) < 4 {
			time.Sleep(2 * time.Millisecond)
		}

		//Assert that the logger was called with the correct error message
		mockLogger.AssertExpectations(t)
		
		//Lets change the logger to avoid panics caused by unexpected log calls to mock loggeer on the teardown
		nilLogger := new(logging.NilLogger)
		wsClient.logger = nilLogger
	})

	t.Run("Terminate connection while writePump runs", func(t *testing.T) {
		//Startup dummy server
		s := &websocketTestServer{}
		s.setup()
		defer s.teardown()
		
		// Create a WebSocket client and connect to the test server
		wsURL := strings.Replace(s.testServer.URL, "http", "ws", 1)
		wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		
		// Initialize the WebsocketClient with the connected WebSocket
		msgHandler := synchronousTestMsgReceiver{receivedMessages: make(chan WebsocketMsg, 1), disconnectAction: make(chan bool, 1)}
		mockLogger := new(logging.MockLogger)
		wsClient := CreateWebsocketClient(&msgHandler, &WebSocketPayloadRegistry{})
		wsClient.conn = wsConn
		wsClient.logger = mockLogger
		wsClient.trafficController = wsUpTrafficController{}

		//Prepare mock logger for the scenario
		mockLogger.On("Infof", "Terminating websocket connection %p",  mock.Anything).Once()
		mockLogger.On("Debugf", "Received done for ws connection %p", mock.Anything).Once()
		mockLogger.On("Debugf", "Handling websocket disconnection of conn %p", mock.Anything).Once()

		//start the pump on another go routine
		go wsClient.writePump()

		//First send a message to know that it goroutine started
		mockLogger.On("Debugf", "Sending WebSocket '%s' msg with transaction Id %s", mock.Anything).Once()
		mockLogger.On("Debugf", "WebSocket msg with transaction Id %s successfully sent",  mock.Anything).Once()
		msg := wrapPayloadInMsg(&WsTestPayload{})
		wsClient.sendChannel <- msg
		for len(mockLogger.Calls) < 2 {
			time.Sleep(2 * time.Millisecond)
		}

		//Terminate the conneection
		wsClient.connCloserGuard.Do(wsClient.closeConnection)

		//We cannot have a smarter way to hold processing the mallformed message
		//since no msg handling will be done
		for len(mockLogger.Calls) < 5 {
			time.Sleep(2 * time.Millisecond)
		}

		//Assert that the logger was called with the correct error message
		mockLogger.AssertExpectations(t)
		
		//Lets change the logger to avoid panics caused by unexpected log calls to mock loggeer on the teardown
		nilLogger := new(logging.NilLogger)
		wsClient.logger = nilLogger
	})
}

func TestWebsocketNotifyDisconnection(t *testing.T) {
	msgHandler := &simpleTestMsgHandler{}
	msgHandleRouter := wsMsgHandlingRouter{msgHandlers: []WebsocketMsgHandler{msgHandler}}
	mockLogger := new(logging.MockLogger)
	wsClient := &WebsocketClient{logger: mockLogger, msgHandleRouter: msgHandleRouter}
	
	//prepare mock logger for scenarion
	mockLogger.On("Debugf", "Handling websocket disconnection of conn %p", mock.Anything).Once()

	//execute
	wsClient.notifyDisconnection()

	//check
	assert.True(t, msgHandler.disconnectRequested)
	mockLogger.AssertExpectations(t)
}

func TestWebsocketCloseConnection(t *testing.T) {
	s := &websocketTestServer{}
	s.setup()
	defer s.teardown()
	
	// Create a WebSocket client and connect to the test server
	wsURL := strings.Replace(s.testServer.URL, "http", "ws", 1)
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	msgHandler := &simpleTestMsgHandler{}
	wsClient := CreateWebsocketClient(msgHandler, &WebSocketPayloadRegistry{})
	
	mockLogger := new(logging.MockLogger)
	wsClient.logger = mockLogger
	wsClient.conn = wsConn

	//prepare mock logger for scenarion
	mockLogger.On("Infof", "Terminating websocket connection %p", mock.Anything).Once()
	mockLogger.On("Debugf", "Handling websocket disconnection of conn %p", mock.Anything).Once()

	//Create a channel for testing channels closure
	closureSyncer := make(chan bool, 3)
	var interruptChannelClosed bool
	_ = interruptChannelClosed
	var sendChannelClosed bool
	_ = sendChannelClosed
	var doneChannelNotified bool
	_ = doneChannelNotified
	var doneChannelClosed bool
	_ = doneChannelClosed

	//Check that done channel will be closed
	go func(){
		closureSyncer <- true
		select {
		case _, ok := <- wsClient.interruptChannel:
			if !ok {
				interruptChannelClosed = true
			}
			break
		}
		closureSyncer <- true
	}()

	//Check that send channel will be closed
	go func() {
		closureSyncer <- true
		select {
		case _, ok := <- wsClient.sendChannel:
			if !ok {
				sendChannelClosed = true
			}
			break
		}
		closureSyncer <- true
	}()

	//Check that done channel will get a done and then close
	go func() {
		closureSyncer <- true
		select {
		case flag, ok := <- wsClient.doneChannel:
			doneChannelNotified = flag

			if ok{
				select {
				case flag, ok = <- wsClient.doneChannel:
					if ok == false{
						doneChannelClosed = true
					}
					break
				}
			}
			
			break
		}
		closureSyncer <- true
	}()

	//execute
	<- closureSyncer
	<- closureSyncer
	<- closureSyncer
	wsClient.closeConnection()
	<- closureSyncer
	<- closureSyncer
	<- closureSyncer

	//check
	assert.True(t, interruptChannelClosed)
	assert.True(t, sendChannelClosed)
	assert.True(t, doneChannelNotified)
	assert.True(t, doneChannelClosed)
	assert.True(t, msgHandler.disconnectRequested)
	mockLogger.AssertExpectations(t)
}

type wsTstErroneousPayload struct {}
func (msg *wsTstErroneousPayload) Type() string { 
	return "networking/wsTstErroneousPayload"
}
func (msg *wsTstErroneousPayload) MarshalJSON() ([]byte, error) {
	
	return []byte{}, fmt.Errorf("wsTstErroneousPayload")
}
func (msg *wsTstErroneousPayload) UnmarshalJSON(data []byte) error {
	return fmt.Errorf("wsTstErroneousPayload")
}


func TestWebsocketClient_sendMsg(t *testing.T) {
	t.Run("Traffic controler rejection", func(t *testing.T) {
		mockLogger := new(logging.MockLogger)
		wsClient := &WebsocketClient{logger: mockLogger, trafficController: wsInactiveTrafficController{}}

		mockLogger.On("Debugf", "Sending WebSocket '%s' msg with transaction Id %s", mock.Anything).Once()
		mockLogger.On("Warnf", "Disregarding '%s' msg with transaction Id %s. Traffic controller replied: %s", mock.Anything).Once()

		msg := wrapPayloadInMsg(&WsTestPayload{})
		res := wsClient.sendMsg(msg)
		
		assert.Equal(t, notAllowedMessageSendError, res)
		mockLogger.AssertExpectations(t)
	})

	t.Run("Json marshalling error", func(t *testing.T) {
		mockLogger := new(logging.MockLogger)
		wsClient := &WebsocketClient{logger: mockLogger, trafficController: wsUpTrafficController{}}

		mockLogger.On("Debugf", "Sending WebSocket '%s' msg with transaction Id %s", mock.Anything).Once()
		mockLogger.On("Errorf", "Failed to marshal WebSocket msg of type %s to json", mock.Anything).Once()

		msg := wrapPayloadInMsg(&wsTstErroneousPayload{})
		res := wsClient.sendMsg(msg)
		
		assert.Equal(t, jsonMarshallingError, res)
		mockLogger.AssertExpectations(t)
	})

	t.Run("Websocket connection error", func(t *testing.T) {
		//Startup dummy server
		s := &websocketTestServer{}
		s.setup()
		defer s.teardown()
		
		// Create a WebSocket client and connect to the test server
		wsURL := strings.Replace(s.testServer.URL, "http", "ws", 1)
		wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		
		//Close the connection for the test scenario
		wsConn.Close()

		// Initialize the WebsocketClient with the connected WebSocket
		msgHandler := synchronousTestMsgReceiver{receivedMessages: make(chan WebsocketMsg, 1), disconnectAction: make(chan bool, 1)}
		mockLogger := new(logging.MockLogger)
		wsClient := CreateWebsocketClient(&msgHandler, &WebSocketPayloadRegistry{})
		wsClient.conn = wsConn
		wsClient.logger = mockLogger
		wsClient.trafficController = wsUpTrafficController{}

		//Prepare mock logger for the scenario
		mockLogger.On("Debugf", "Sending WebSocket '%s' msg with transaction Id %s", mock.Anything).Once()
		mockLogger.On("Errorf", "Failed to send WebSocket msg %s through %p",  mock.Anything).Once()

		msg := wrapPayloadInMsg(&WsTestPayload{})
		res := wsClient.sendMsg(msg)
		
		assert.Equal(t, wsSendInternalError, res)
		mockLogger.AssertExpectations(t)
	})

	t.Run("Successful send", func(t *testing.T) {
		//Startup dummy server
		s := &websocketTestServer{}
		s.setup()
		defer s.teardown()
		
		// Create a WebSocket client and connect to the test server
		wsURL := strings.Replace(s.testServer.URL, "http", "ws", 1)
		wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		
		// Initialize the WebsocketClient with the connected WebSocket
		msgHandler := synchronousTestMsgReceiver{receivedMessages: make(chan WebsocketMsg, 1), disconnectAction: make(chan bool, 1)}
		mockLogger := new(logging.MockLogger)
		wsClient := CreateWebsocketClient(&msgHandler, &WebSocketPayloadRegistry{})
		wsClient.conn = wsConn
		wsClient.logger = mockLogger
		wsClient.trafficController = wsUpTrafficController{}

		//Prepare mock logger for the scenario
		mockLogger.On("Debugf", "Sending WebSocket '%s' msg with transaction Id %s", mock.Anything).Once()
		mockLogger.On("Debugf", "WebSocket msg with transaction Id %s successfully sent",  mock.Anything).Once()

		msg := wrapPayloadInMsg(&WsTestPayload{})
		res := wsClient.sendMsg(msg)
		
		assert.Equal(t, msgSuccessfullySent, res)
		mockLogger.AssertExpectations(t)
	})
}