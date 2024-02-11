package networking

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/agaraleas/DecentralizedNetworkSync/logging"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	websocketMsgSendQueueSize = 10
)

// WebsocketMsg: Represenets a websocket message that a WebsocketClient can handle
type WebsocketMsg struct {
	TransactionId uuid.UUID           `json:"transactionId"`
	MsgType       string              `json:"msgType"`
	Payload       WebSocketMsgPayload `json:"payload"`
}

// WebSocketMsgPayload: Represents the custom part of a websocket message that will be sent or read
// from a remote peer. Should implment basic json marhsall interface
type WebSocketMsgPayload interface {
	Type() string
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
}

// WebSocketPayloadRegistry: Represent a deposit of message templates that a Websocket Client can handle
type WebSocketPayloadRegistry struct {
	payloadTemplates map[string]*WebSocketMsgPayload
}

func (r *WebSocketPayloadRegistry) Register(payload WebSocketMsgPayload) {
	if r.payloadTemplates == nil {
		r.payloadTemplates = make(map[string]*WebSocketMsgPayload)
	}

	key := payload.Type()
	r.payloadTemplates[key] = &payload
}

func (r *WebSocketPayloadRegistry) getTemplate(key string) (WebSocketMsgPayload, bool) {
	payload, found := r.payloadTemplates[key]
	if found {
		return *payload, true
	} else {
		return nil, false
	}
}

// WebsocketMsgHandler: Represents the object which will do the actual handling when a WebsocketMsg wil
// be received from a remote peer. Offers also a slot to be called upon disconnection
type WebsocketMsgHandler interface {
	Handle(ws *WebsocketClient, msg WebsocketMsg) error
	Disconnecting(ws *WebsocketClient)
}

// websocketDialer: Abstraction of the dialer to allo mocking for testing purposes
type websocketDialer interface {
	Dial(urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error)
}

// wsMsgHandlingRouter: Dispatches a received message to appropriate handler (custom / internal)
type wsMsgHandlingRouter struct {
	msgHandlers       []WebsocketMsgHandler
	payloadRegistries []*WebSocketPayloadRegistry
}

func (h *wsMsgHandlingRouter) dispatch(rawMsg []byte, msgType string, ws *WebsocketClient) error {
	for i := 0; i <= maxMsgHandlerIndex; i += 1 {
		payloadRegistry := h.payloadRegistries[i]
		payload, found := payloadRegistry.getTemplate(msgType)

		if found {
			msg := WebsocketMsg{Payload: payload}
			err := json.Unmarshal(rawMsg, &msg)
			if err != nil {
				ws.logger.Errorf("Failed to unmarshall websocket message %s", rawMsg)
				return fmt.Errorf("Failed to unmarshall websocket message")
			}

			allowed, err := ws.trafficController.isAllowed(msg)
			if !allowed {
				ws.logger.Warnf("Will not handle received msg '%s' due to Traffic controller. %s", err.Error())
				return err
			}

			msgHandler := h.msgHandlers[i]
			return msgHandler.Handle(ws, msg)
		}
	}

	ws.logger.Warnf("Failed to handle ws message. Type '%s' unknown to all payload registries", msgType)
	return fmt.Errorf("Failed to handle ws message. Type '%s' unknown to all payload registries", msgType)
}

const (
	customMsgHandlerIndex   = 0
	internalMsgHandlerIndex = 1
	maxMsgHandlerIndex      = 1
)

// wsInternalMsgHandler: Handles internal websocket messages such as Hello, Goodbye etc
type wsInternalMsgHandler struct {
	goodbyeMsgAckWaiter chan struct{}
}

func (h *wsInternalMsgHandler) Handle(ws *WebsocketClient, msg WebsocketMsg) error {
	if _, isGoodbyeMsg := msg.Payload.(*wsGoodByeMsg); isGoodbyeMsg {
		ws.disconnectNotifyGuard.Do(ws.notifyDisconnection)
		ws.Send(new(wsGoodByeMsgAck))
	} else if _, isGoodbyeMsgAck := msg.Payload.(*wsGoodByeMsgAck); isGoodbyeMsgAck {
		close(h.goodbyeMsgAckWaiter)
	} else {
		ws.logger.Warnf("Internal msg handler called to handle msg with type %s", msg.MsgType)
	}

	return nil
}
func (h *wsInternalMsgHandler) Disconnecting(ws *WebsocketClient) {}

// wsGoodByeMsg: internal websocket message to notify remote peer for going away
type wsGoodByeMsg struct{}

func (msg *wsGoodByeMsg) Type() string {
	return "networking/wsGoodByeMsg"
}
func (msg *wsGoodByeMsg) MarshalJSON() ([]byte, error) {
	JSON := `{}`
	return []byte(JSON), nil
}
func (msg *wsGoodByeMsg) UnmarshalJSON(data []byte) error {
	return nil
}

// wsGoodByeMsg: internal websocket message to acknowledge disconnection of a remote peer
type wsGoodByeMsgAck struct{}

func (msg *wsGoodByeMsgAck) Type() string {
	return "networking/wsGoodByeMsgAck"
}
func (msg *wsGoodByeMsgAck) MarshalJSON() ([]byte, error) {
	JSON := `{}`
	return []byte(JSON), nil
}
func (msg *wsGoodByeMsgAck) UnmarshalJSON(data []byte) error {
	return nil
}

// wsTrafficController: controls which messages are allowed to be sent
type wsTrafficController interface {
	isAllowed(msg WebsocketMsg) (bool, error)
	isDisconnectionRequested() bool
}

// wsInactiveTrafficController: implements wsTrafficController when no connection is yet established
type wsInactiveTrafficController struct{}

func (c wsInactiveTrafficController) isAllowed(msg WebsocketMsg) (bool, error) {
	return false, fmt.Errorf("Websocket connection not established")
}

func (c wsInactiveTrafficController) isDisconnectionRequested() bool {
	return false
}

// wsUpTrafficController: implements wsTrafficController when connection is established
type wsUpTrafficController struct{}

func (c wsUpTrafficController) isAllowed(msg WebsocketMsg) (bool, error) {
	return true, nil
}

func (c wsUpTrafficController) isDisconnectionRequested() bool {
	return false
}

// wsUpTrafficController: implements wsTrafficController when connection is established
type wsDisconnectingTrafficController struct{}

func (c wsDisconnectingTrafficController) isAllowed(msg WebsocketMsg) (bool, error) {
	allowed := msg.MsgType == typeOfPayload(&wsGoodByeMsg{}) ||
		msg.MsgType == typeOfPayload(&wsGoodByeMsgAck{})

	if allowed {
		return true, nil
	} else {
		return false, fmt.Errorf("Msg type '%s' is disregarded during disconnection", msg.MsgType)
	}
}

func (c wsDisconnectingTrafficController) isDisconnectionRequested() bool {
	return true
}

// typeOfPayload: return type of payload as a string for readability purposes
func typeOfPayload(payload WebSocketMsgPayload) string {
	return payload.Type()
}

// AbstractWebsocketClient: Defines methods for interacting with a WebSocketClient.
type AbstractWebsocketClient interface {
	Connect(serverUrl string) error
	Disconnect()
	Send(payload WebSocketMsgPayload)
	Reply(payload WebSocketMsgPayload, rx WebsocketMsg)
}

// WebsocketClient: Represents the facade which handles websocket communication to a remote peer
type WebsocketClient struct {
	msgHandleRouter       wsMsgHandlingRouter
	conn                  *websocket.Conn
	dialer                websocketDialer
	logger                logging.AbstractLogger
	trafficController     wsTrafficController
	interruptChannel      chan os.Signal
	sendChannel           chan WebsocketMsg
	doneChannel           chan bool
	disconnectNotifyGuard sync.Once
	connCloserGuard       sync.Once
}

func createMsgHandlingRouterForWsClient(customMsgHandler WebsocketMsgHandler,
	customPayloadRegistry *WebSocketPayloadRegistry) wsMsgHandlingRouter {

	msgHandlerRouter := wsMsgHandlingRouter{msgHandlers: make([]WebsocketMsgHandler, 2),
		payloadRegistries: make([]*WebSocketPayloadRegistry, 2),
	}

	msgHandlerRouter.msgHandlers[customMsgHandlerIndex] = customMsgHandler
	msgHandlerRouter.payloadRegistries[customMsgHandlerIndex] = customPayloadRegistry

	var internalPayloadRegistry WebSocketPayloadRegistry
	internalPayloadRegistry.Register(new(wsGoodByeMsg))
	internalPayloadRegistry.Register(new(wsGoodByeMsgAck))

	msgHandlerRouter.payloadRegistries[internalMsgHandlerIndex] = &internalPayloadRegistry
	msgHandlerRouter.msgHandlers[internalMsgHandlerIndex] = &wsInternalMsgHandler{goodbyeMsgAckWaiter: make(chan struct{})}

	return msgHandlerRouter
}

func CreateWebsocketClient(msgHandler WebsocketMsgHandler,
	payloadRegistry *WebSocketPayloadRegistry) *WebsocketClient {

	wsClient := &WebsocketClient{msgHandleRouter: createMsgHandlingRouterForWsClient(msgHandler, payloadRegistry),
		interruptChannel:  make(chan os.Signal, 1),
		sendChannel:       make(chan WebsocketMsg, websocketMsgSendQueueSize),
		doneChannel:       make(chan bool),
		dialer:            websocket.DefaultDialer,
		logger:            logging.Log,
		trafficController: wsInactiveTrafficController{},
	}

	signal.Notify(wsClient.interruptChannel, os.Interrupt, syscall.SIGTERM)
	signal.Notify(wsClient.interruptChannel, os.Interrupt, syscall.SIGKILL)
	signal.Notify(wsClient.interruptChannel, os.Interrupt, syscall.SIGINT)

	return wsClient
}

func (ws *WebsocketClient) Connect(serverUrl string) error {
	u, err := url.Parse(serverUrl)
	if err != nil {
		ws.logger.Errorf("WebsocketClient::Connect Failed to parse url %s", serverUrl)
		return err
	}

	conn, _, err := ws.dialer.Dial(u.String(), nil)

	if err != nil {
		ws.logger.Errorf("WebsocketClient::Connect Failed to dial url %s", u.String())
		return err
	}

	ws.logger.Debugf("WebsocketClient::Connect Successfully established connection %p to %s", conn, u.String())
	ws.trafficController = wsUpTrafficController{}
	ws.conn = conn
	return nil
}

func waitForGoodbyeAck(waitChann chan struct{}, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)

	select {
	case <-waitChann:
		timer.Stop()
		return true
	case <-timer.C:
		return false
	}
}

func scheduleForcefulDisconnection(ws *WebsocketClient, timeout time.Duration) {
	//Disconnect forcefully if other peer does not acknowledge closing of connection after 1 sec
	//Since disconnectin is protected by Once syncer, I will shedule a disconnetion which
	//will run only if readPump does not receive a graceful connection close in the meantime

	timerChan := time.After(1 * time.Second)

	go func() {
		<-timerChan
		ws.connCloserGuard.Do(ws.closeConnection)
	}()
}

func closeWsConnectionGracefully(conn *websocket.Conn) {
	closeMessage := websocket.FormatCloseMessage(websocket.CloseGoingAway, "")
	conn.WriteMessage(websocket.CloseMessage, closeMessage)
}

func (ws *WebsocketClient) Disconnect() {
	ws.trafficController = wsDisconnectingTrafficController{}

	goodbyeMsg := new(wsGoodByeMsg)
	ws.Send(goodbyeMsg)

	internalMsgHandler := ws.msgHandleRouter.msgHandlers[internalMsgHandlerIndex].(*wsInternalMsgHandler)
	acknowledged := waitForGoodbyeAck(internalMsgHandler.goodbyeMsgAckWaiter, 5*time.Second)

	ws.disconnectNotifyGuard.Do(ws.notifyDisconnection)

	scheduleForcefulDisconnection(ws, 1*time.Second)
	if acknowledged {
		closeWsConnectionGracefully(ws.conn)
	}
}

func (ws *WebsocketClient) Send(payload WebSocketMsgPayload) {
	msg := WebsocketMsg{TransactionId: uuid.New(),
		MsgType: payload.Type(),
		Payload: payload,
	}
	ws.sendChannel <- msg
}

func (ws *WebsocketClient) Reply(payload WebSocketMsgPayload, rx WebsocketMsg) {
	msg := WebsocketMsg{TransactionId: rx.TransactionId,
		MsgType: payload.Type(),
		Payload: payload,
	}
	ws.sendChannel <- msg
}

func (ws *WebsocketClient) handleReceivedMsg(txt []byte) {
	var typeReader struct {
		Value string `json:"msgType"`
	}

	err := json.Unmarshal(txt, &typeReader)
	if err != nil {
		ws.logger.Errorf("Failed to unmarshall type of websocket message %s", txt)
		return
	}

	ws.msgHandleRouter.dispatch(txt, typeReader.Value, ws)
}

func (ws *WebsocketClient) readPump() {
	defer func() {
		ws.connCloserGuard.Do(ws.closeConnection)
	}()

	for {
		wsMsgType, txt, err := ws.conn.ReadMessage()
		if err != nil {
			if ws.trafficController.isDisconnectionRequested() {
				ws.logger.Debugf("Websocket connection %p closed due to imminent disconnection", ws.conn)
				break
			} else if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				ws.logger.Debugf("Remote peer closed ws connection %p", ws.conn)
				break
			} else {
				ws.logger.Errorf("Failed to read Websocket Message from connection %p. Error: %v", ws.conn, err)
				break
			}
		}

		switch wsMsgType {
		case websocket.TextMessage:
			go ws.handleReceivedMsg(txt)
		case websocket.CloseMessage:
			ws.logger.Debugf("Received close message for ws connection %p from remote peer", ws.conn)
			break
		}
	}
}

func (ws *WebsocketClient) writePump() {
	defer func() {
		ws.connCloserGuard.Do(ws.closeConnection)
	}()

	for {
		select {
		case msg, ok := <-ws.sendChannel:
			if !ok {
				ws.logger.Debugf("Received not ok in send channel of WebSocket client for ws connection %p", ws.conn)
				return
			}

			res := ws.sendMsg(msg)
			if res == wsSendInternalError {
				ws.logger.Debugf("Stopping write pump of WebSocket client for ws connection %p", ws.conn)
				return
			}

		case <-ws.interruptChannel:
		case <-ws.doneChannel:
			ws.logger.Debugf("Received done for ws connection %p", ws.conn)
			return
		}
	}
}

func (ws *WebsocketClient) notifyDisconnection() {
	ws.logger.Debugf("Handling websocket disconnection of conn %p", ws.conn)

	for _, msgHandler := range ws.msgHandleRouter.msgHandlers {
		msgHandler.Disconnecting(ws)
	}
}

func (ws *WebsocketClient) closeConnection() {
	ws.logger.Infof("Terminating websocket connection %p", ws.conn)
	ws.disconnectNotifyGuard.Do(ws.notifyDisconnection)
	ws.doneChannel <- true
	close(ws.doneChannel)
	close(ws.interruptChannel)
	close(ws.sendChannel)
	ws.conn.Close()
}

type msgSendingResult int

const (
	msgSuccessfullySent msgSendingResult = iota
	notAllowedMessageSendError
	jsonMarshallingError
	wsSendInternalError
)

func (ws *WebsocketClient) sendMsg(msg WebsocketMsg) msgSendingResult {
	ws.logger.Debugf("Sending WebSocket '%s' msg with transaction Id %s", msg.MsgType, msg.TransactionId)

	allowed, err := ws.trafficController.isAllowed(msg)
	if !allowed {
		ws.logger.Warnf("Disregarding '%s' msg with transaction Id %s. Traffic controller replied: %s", err.Error())
		return notAllowedMessageSendError
	}

	json, err := json.Marshal(msg)
	if err != nil {
		ws.logger.Errorf("Failed to marshal WebSocket msg of type %s to json", msg.MsgType)
		return jsonMarshallingError
	}

	err = ws.conn.WriteMessage(websocket.TextMessage, []byte(string(json)))
	if err != nil {
		ws.logger.Errorf("Failed to send WebSocket msg %s through %p", msg.TransactionId, ws.conn)
		return wsSendInternalError
	}

	ws.logger.Debugf("WebSocket msg with transaction Id %s successfully sent", msg.TransactionId)
	return msgSuccessfullySent
}
