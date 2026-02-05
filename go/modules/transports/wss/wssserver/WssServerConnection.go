package wssserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
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
	transports.ServerConnectionBase

	// connection ID
	//connectionID string

	// clientID is the account ID of the connected client
	// clientID string

	// connection request remote address
	httpReq *http.Request

	// isConnected atomic.Bool

	// track last used time to auto-close stale connections
	lastActivity time.Time

	// mutex for controlling writing and closing
	// mux sync.RWMutex

	// converter for request/response messages
	messageConverter transports.IMessageConverter

	// notifHandler handles the requests received from the remote producer
	notifHandler msg.NotificationHandler

	// reqHandler handles the requests received from the remote consumer
	reqHandler msg.RequestHandler

	// request-response channel
	rnrChan *msg.RnRChan

	// how long to wait for a response after sending a request
	respTimeout time.Duration

	// underlying websocket connection
	wssConn *websocket.Conn
}

// _send sends the seriaziled websocket message to the connected client
func (sc *WssServerConnection) _send(msg []byte) (err error) {

	if !sc.IsConnected() {
		err = fmt.Errorf(
			"_send: connection with client '%s' is now closed", sc.GetClientID())
		slog.Warn(err.Error())
	} else {
		// websockets do not allow concurrent write
		sc.Mux.Lock()
		defer sc.Mux.Unlock()
		err = sc.wssConn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			err = fmt.Errorf("WssServerConnection._send write error: %s", err)
		}
	}
	return err
}

// Close closes the connection and ends the read loop
func (sc *WssServerConnection) Close() {
	sc.Mux.Lock()
	defer sc.Mux.Unlock()
	if sc.IsConnected() {
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

// onMessage handles an incoming websocket message
// The message is converted into a request, response or notification and passed
// on to the registered handler.
func (sc *WssServerConnection) onMessage(raw []byte) {
	sc.Mux.Lock()
	sc.lastActivity = time.Now()
	sc.Mux.Unlock()
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

// server connection receives a notification from remote client (producer).
// pass it on to the registered (upstream) notification handler.
//
// remote producer [notification] -> client<=>server -> onNotification
func (sc *WssServerConnection) onNotification(notif *msg.NotificationMessage) {
	sc.notifHandler(notif)
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
		err = sc.reqHandler(req, func(reply *msg.ResponseMessage) error {
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

	// this responsehandler points to the rnrChannel that matches the correlationID to the replyTo handler
	handled := sc.rnrChan.HandleResponse(resp, sc.respTimeout)
	if !handled {
		slog.Warn("onResponse: No response handler for request, response is lost",
			"correlationID", resp.CorrelationID,
			"op", resp.Operation,
			"thingID", resp.ThingID,
			"name", resp.Name)
	}
}

// ReadLoop reads incoming websocket messages in a loop, until connection closes or context is cancelled
func (sc *WssServerConnection) ReadLoop(ctx context.Context, wssConn *websocket.Conn) {

	// close the client when the context ends drops
	go func() {
		select {
		case <-ctx.Done(): // remote client connection closed
			slog.Debug("WssServerConnection.ReadLoop: Remote client disconnected")
			// close channel when no-one is writing
			// in the meantime keep reading to prevent deadlock
			_ = wssConn.Close()
		}
	}()
	// read messages from the client until the connection closes
	for sc.IsConnected() { // sseMsg := range sseChan {
		_, raw, err := wssConn.ReadMessage()
		if err != nil {
			// ending the read loop and returning will close the connection
			break
		}
		// process the message in the background to allow concurrent messages
		go sc.onMessage(raw)
	}
}

// SendNotification sends a notification to the client if subscribed.
func (sc *WssServerConnection) SendNotification(notif *msg.NotificationMessage) {

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
		if err != nil {
			// maybe the connection dropped. It should have been removed though so something went wrong.
			slog.Warn("SendNotification: Unable to send the notification to the client",
				"clientID", sc.ClientID,
				"err", err.Error())
		}
	}
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
	sc.rnrChan.WaitWithCallback(req.CorrelationID, sc.respTimeout, responseHandler)

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

// SetTimeout set the timeout sending requests
func (sc *WssServerConnection) SetTimeout(timeout time.Duration) {
	sc.respTimeout = timeout
}

// NewWSSServerConnection creates a new Websocket connection instance for use by
// agents and consumers.
// This implements the IConnection interface.
//
// clientID is the consumer or agent authenticated ID
// r is the request used to establish this connection
// wssConn is the connection on which to send/receive messages to the client
// messageConverter maps protocol messages to standard RRN
// reqHandler will handle incoming request messages (required)
// notifHandler will handling incoming notification messages (required)
func NewWSSServerConnection(
	clientID string, r *http.Request,
	wssConn *websocket.Conn,
	messageConverter transports.IMessageConverter,
	reqHandler msg.RequestHandler,
	notifHandler msg.NotificationHandler,
) *WssServerConnection {

	cid := "WSS" + shortid.MustGenerate()
	if reqHandler == nil || notifHandler == nil {
		panic("WSS incoming connection needs request and notification handlers.")
	}

	c := &WssServerConnection{
		wssConn:          wssConn,
		messageConverter: messageConverter,
		httpReq:          r,
		lastActivity:     time.Time{},
		rnrChan:          msg.NewRnRChan(),
		respTimeout:      transports.DefaultRpcTimeout,
		reqHandler:       reqHandler,
		notifHandler:     notifHandler,
	}
	c.Init(clientID, r.URL.String(), cid)
	return c
}
