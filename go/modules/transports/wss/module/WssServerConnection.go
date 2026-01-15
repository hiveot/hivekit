package module

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/teris-io/shortid"
)

type WSSMessage map[string]any

// WssServerConnection is  the server side instance of a connection by a client.
// This implements the IServerConnection interface for sending messages to
// agent or consumers.
type WssServerConnection struct {
	transports.ConnectionBase

	// connection ID
	//connectionID string

	// clientID is the account ID of the connected client
	clientID string

	// connection request remote address
	httpReq *http.Request

	// isConnected atomic.Bool

	// track last used time to auto-close inactive cm
	lastActivity time.Time

	// mutex for controlling writing and closing
	mux sync.RWMutex

	// notify client of a connect or disconnect
	connectionHandlerPtr atomic.Pointer[transports.ConnectionHandler]

	// converter for request/response messages
	messageConverter transports.IMessageConverter

	// underlying websocket connection
	wssConn *websocket.Conn

	// request-response channel
	rnrChan *msg.RnRChan

	// how long to wait for a response after sending a request
	respTimeout time.Duration

	// module sink that handles the incoming messages
	sink modules.IHiveModule
}

// _send sends the seriaziled websocket message to the connected client
func (sc *WssServerConnection) _send(msg []byte) (err error) {

	if !sc.IsConnected() {
		err = fmt.Errorf(
			"_send: connection with client '%s' is now closed", sc.GetClientID())
		slog.Warn(err.Error())
	} else {
		// websockets do not allow concurrent write
		sc.mux.Lock()
		defer sc.mux.Unlock()
		err = sc.wssConn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			err = fmt.Errorf("WssServerConnection._send write error: %s", err)
		}
	}
	return err
}

// Close closes the connection and ends the read loop
func (sc *WssServerConnection) Close() {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	if sc.IsConnected() {
		sc.onConnection(false, nil)
		_ = sc.wssConn.Close()
	}
}

// // HasSubscription returns true if this connection has subscribed to the given notification
// func (sc *WssServerConnection) HasSubscription(notif *msg.NotificationMessage) bool {
// 	switch notif.Operation {

// 	case wot.OpSubscribeEvent, wot.OpSubscribeAllEvents:
// 		correlationID := sc.subscriptions.GetSubscription(notif.ThingID, notif.Name)
// 		if correlationID != "" {
// 			slog.Info("HasSubscription (event subscription)",
// 				slog.String("clientID", sc.cinfo.ClientID),
// 				slog.String("thingID", notif.ThingID),
// 				slog.String("event name", notif.Name),
// 			)
// 			return true
// 		}
// 	case wot.OpObserveProperty, wot.OpObserveMultipleProperties, wot.OpObserveAllProperties:
// 		correlationID := sc.observations.GetSubscription(notif.ThingID, notif.Name)
// 		if correlationID != "" {
// 			slog.Info("HasSubscription (observed property(ies))",
// 				slog.String("clientID", sc.cinfo.ClientID),
// 				slog.String("thingID", notif.ThingID),
// 				slog.String("name", notif.Name),
// 			)
// 			return true
// 		}
// 	case wot.OpInvokeAction:
// 		// action progress update, for original sender only
// 		slog.Info("HasSubscription (action status)",
// 			slog.String("clientID", sc.cinfo.ClientID),
// 			slog.String("thingID", notif.ThingID),
// 			slog.String("name", notif.Name),
// 		)
// 		return true
// 	default:
// 		slog.Warn("Unknown notification: " + notif.Operation)
// 	}
// 	return false
// }

// IsConnected returns the connection status
//
//	func (sc *WssServerConnection) IsConnected() bool {
//		return sc.isConnected.Load()
//	}
func (sc *WssServerConnection) onConnection(connected bool, err error) {
	if !connected {
		sc.ConnectionBase.Disconnect()
	}
	chPtr := sc.connectionHandlerPtr.Load()
	if chPtr != nil {
		(*chPtr)(connected, err, sc)
	}
}

// onMessage handles an incoming websocket message
// The message is converted into a request, response or notification and passed
// on to the registered handler.
func (sc *WssServerConnection) onMessage(raw []byte) {
	sc.mux.Lock()
	sc.lastActivity = time.Now()
	sc.mux.Unlock()
	var notif *msg.NotificationMessage
	var req *msg.RequestMessage
	var resp *msg.ResponseMessage

	// the only way to know which message type it is is to decode it
	notif = sc.messageConverter.DecodeNotification(raw)
	if notif != nil {
		// sender is identified by the server, not the client
		notif.SenderID = sc.GetClientID()
		sc.onNotification(notif)
		return
	}
	resp = sc.messageConverter.DecodeResponse(raw)
	if resp != nil {
		// sender is identified by the server, not the client
		resp.SenderID = sc.GetClientID()
		sc.onResponse(resp)
		return
	}
	req = sc.messageConverter.DecodeRequest(raw)
	if req != nil {
		// sender is identified by the server, not the client
		req.SenderID = sc.GetClientID()
		sc.onRequest(req)
		return
	}
	slog.Warn("onMessage: Message is not a notification, request or response")
}

// onNotification is passed the received notification message to the module sink
func (sc *WssServerConnection) onNotification(notif *msg.NotificationMessage) {
	sc.sink.HandleNotification(notif)
}

// onRequest is passed the received request message
// This method handles subscriptions using the ConnectionBase.
func (sc *WssServerConnection) onRequest(req *msg.RequestMessage) {
	var resp *msg.ResponseMessage
	var err error

	switch req.Operation {
	case wot.HTOpPing:
		resp = req.CreateResponse("pong", nil)

	case wot.OpSubscribeEvent, wot.OpSubscribeAllEvents:
		sc.SubscribeEvent(req.ThingID, req.Name, req.CorrelationID)
		resp = req.CreateResponse(nil, nil)

	case wot.OpUnsubscribeEvent, wot.OpUnsubscribeAllEvents:
		sc.UnsubscribeEvent(req.ThingID, req.Name)
		resp = req.CreateResponse(nil, nil)

	case wot.OpObserveProperty, wot.OpObserveAllProperties:
		sc.ObserveProperty(req.ThingID, req.Name, req.CorrelationID)
		resp = req.CreateResponse(nil, nil)

	case wot.OpUnobserveProperty, wot.OpUnobserveAllProperties:
		sc.UnobserveProperty(req.ThingID, req.Name)
		resp = req.CreateResponse(nil, nil)
	default:
		// this is not a subscription to notifications so forward it to the module sink
		// response will be handled asynchronously
		err = sc.sink.HandleRequest(req, func(reply *msg.ResponseMessage) error {
			// the callback is async so handle it separately
			if reply != nil {
				err = sc.SendResponse(reply)
			} else {
				slog.Error("onRequest: Sink response callback without response")
			}
			return err
		})
		// if handling the request failed, return an error response
		if err != nil {
			resp = req.CreateErrorResponse(err)
		}
	}
	if resp != nil {
		err = sc.SendResponse(resp)
	}
	if err != nil {
		slog.Warn("Error handling request message", "err", err.Error())
	}
}

// onResponse is passed the received response message after decoding to the standard response message format.
//
// This passes the response to the RNR response handler to serve any request handlers that are waiting
// for a response. All this takes place asynchronously without blocking the connection.
//
// If the RNR handler doesn't have a matching correlationID listed then the response is passed to the
// connection response handler.
func (sc *WssServerConnection) onResponse(resp *msg.ResponseMessage) {
	var err error
	// this responsehandler points to the rnrChannel that matches the correlationID to the replyTo handler
	handled := sc.rnrChan.HandleResponse(resp)
	if !handled {
		err = sc.sink.HandleResponse(resp)
	}
	if err != nil {
		slog.Warn("Error handling response message", "err", err.Error())
	}
}

// ReadLoop reads incoming websocket messages in a loop, until connection closes or context is cancelled
func (sc *WssServerConnection) ReadLoop(ctx context.Context, wssConn *websocket.Conn) {

	//var readLoop atomic.Bool
	sc.onConnection(true, nil)

	// close the client when the context ends drops
	go func() {
		select {
		case <-ctx.Done(): // remote client connection closed
			slog.Debug("WssServerConnection.ReadLoop: Remote client disconnected")
			// close channel when no-one is writing
			// in the meantime keep reading to prevent deadlock
			_ = wssConn.Close()
			sc.onConnection(false, nil)
		}
	}()
	// read messages from the client until the connection closes
	for sc.IsConnected() { // sseMsg := range sseChan {
		_, raw, err := wssConn.ReadMessage()
		if err != nil {
			// avoid further writes
			sc.onConnection(false, err)
			// ending the read loop and returning will close the connection
			break
		}
		// process the message in the background to allow concurrent messages
		go sc.onMessage(raw)
	}
}

// SendNotification sends a notification to the client if subscribed.
func (sc *WssServerConnection) SendNotification(notif *msg.NotificationMessage) (err error) {

	slog.Info("SendNotification",
		slog.String("clientID", sc.GetClientID()),
		slog.String("thingID", notif.ThingID),
		slog.String("op", notif.Operation),
		slog.String("name", notif.Name),
	)
	if sc.HasSubscription(notif) {
		msg, err := sc.messageConverter.EncodeNotification(notif)
		if err == nil {
			err = sc._send(msg)
		}
	}
	return err
}

// SendRequest sends the request to the client (agent).
//
// This accepts a response handler through which the response is received. If not
// provided then the response will be forwarded to the module sink.
//
// Intended to be used by gateways that forward requests from consumers to agents, where the agent
// has connected to the gateway using (connection reversal) and the gateway proxies on behalf of the consumer.
//
// When a response is received it is passed to the replyTo handler.
//
// If this returns an error then no request was sent.
func (sc *WssServerConnection) SendRequest(
	req *msg.RequestMessage, responseHandler msg.ResponseHandler) error {

	wssMsg, err := sc.messageConverter.EncodeRequest(req)
	if err != nil {
		return err
	}
	// without a replyTo simply send the request
	if responseHandler == nil {
		err = sc._send(wssMsg)
		return err
	}

	// with a replyTo, send the response async to the replyTo handler
	// catch the response in a channel linked by correlation-id
	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	// the websocket connection response handlers will convert the message and pass it to the RNR channels
	sc.rnrChan.WaitWithCallback(req.CorrelationID, responseHandler)

	// now the RNR channel is ready, send the request message
	err = sc._send(wssMsg)
	return err
}

// SendResponse sends a response to the remote client.
// If this returns an error then no response was sent.
func (sc *WssServerConnection) SendResponse(resp *msg.ResponseMessage) (err error) {

	//slog.Info("SendResponse (server->client)",
	//	slog.String("clientID", sc.cinfo.ClientID),
	//	slog.String("correlationID", resp.CorrelationID),
	//	slog.String("operation", resp.Operation),
	//	slog.String("name", resp.Name),
	//	slog.String("status", resp.Status),
	//	slog.String("type", resp.MessageType),
	//	slog.String("senderID", resp.SenderID),
	//)

	msg, _ := sc.messageConverter.EncodeResponse(resp)
	err = sc._send(msg)
	return err
}

func (sc *WssServerConnection) SetConnectHandler(cb transports.ConnectionHandler) {
	if cb == nil {
		sc.connectionHandlerPtr.Store(nil)
	} else {
		sc.connectionHandlerPtr.Store(&cb)
	}
}

// NewWSSServerConnection creates a new Websocket connection instance for use by
// agents and consumers.
// This implements the IServerConnection interface.
// The sink (required) will received incoming messages.
func NewWSSServerConnection(
	clientID string, r *http.Request,
	wssConn *websocket.Conn,
	messageConverter transports.IMessageConverter,
	sink modules.IHiveModule,
) *WssServerConnection {

	cid := "WSS" + shortid.MustGenerate()
	if sink == nil {
		slog.Error("WSS incoming connection but no sink is provided. Messages will NOT be handled.")
	}

	c := &WssServerConnection{
		wssConn:          wssConn,
		clientID:         clientID,
		messageConverter: messageConverter,
		httpReq:          r,
		lastActivity:     time.Time{},
		mux:              sync.RWMutex{},
		rnrChan:          msg.NewRnRChan(transports.DefaultRpcTimeout),
		respTimeout:      transports.DefaultRpcTimeout,
		sink:             sink,
	}
	c.Init(clientID, r.URL.String(), cid)
	return c
}
