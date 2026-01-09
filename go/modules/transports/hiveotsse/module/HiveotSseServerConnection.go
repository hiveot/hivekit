package module

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
)

type SSEEvent struct {
	EventType string // type of message, e.g. event, action or other
	Payload   string // message content
}

// SSEPingEvent can be used by the server to ping the client that the connection is ready
const SSEPingEvent = "sse-ping"

// HiveotSseServerConnection handles the SSE connection by remote client
//
// The Sse-sc protocol binding uses a 'hiveot' message envelope for sending messages
// between server and consumer.
//
// This implements the IServerConnection interface for sending messages to
// the client over SSE.
type HiveotSseServerConnection struct {
	// Connection information such as clientID, cid, address, protocol etc
	transports.ConnectionBase

	//// connection ID (from header, without clientID prefix)
	//connectionID string
	//
	//// clientID is the account ID of the agent or consumer
	//clientID string

	// connection remote address
	remoteAddr string

	// incoming sse request
	httpReq *http.Request

	// isConnected atomic.Bool

	// track last used time to auto-close inactive cm
	lastActivity time.Time

	// mutex for controlling writing and closing
	mux sync.RWMutex

	// notify client of a connect or disconnect
	connectionHandler transports.ConnectionHandler
	// handler for requests send by clients
	appRequestHandlerPtr atomic.Pointer[msg.RequestHandler]
	// handler for responses sent by agents
	responseHandlerPtr atomic.Pointer[msg.ResponseHandler]
	// handler for notifications sent by agents
	notificationHandlerPtr atomic.Pointer[msg.NotificationHandler]

	sseChan chan SSEEvent

	// subscriptions transports.Subscriptions
	// observations  transports.Subscriptions
	//
	correlData map[string]chan any

	// the Request-and-Response helper that links responses from http
	// with requests send over sse.
	// This instance is owned by the http server which passes responses to it.
	rnrChan *transports.RnRChan
	// timeout to observe when waiting for responses
	respTimeout time.Duration
}

//type HttpActionStatus struct {
//	CorrelationID string `json:"request_id"`
//	ThingID   string `json:"thingID"`
//	Name      string `json:"name"`
//	Data      any    `json:"data"`
//	Error     string `json:"error"`
//}

// _send sends a request, response or notification message to the client over SSE.
// This is different from the WoT SSE subprotocol in that the payload is the
// message envelope and can carry any operation.
func (sc *HiveotSseServerConnection) _send(msgType string, msg any) (err error) {

	payloadJSON, _ := jsoniter.MarshalToString(msg)
	sseMsg := SSEEvent{
		EventType: msgType,
		Payload:   payloadJSON,
	}
	sc.mux.Lock()
	defer sc.mux.Unlock()
	if sc.IsConnected() {
		slog.Debug("_send",
			slog.String("to", sc.GetClientID()),
			slog.String("MessageType", msgType),
		)
		sc.sseChan <- sseMsg
	}
	// as long as the channel exists, delivery will take place
	return nil
}

// Close closes the connection and ends the read loop
func (sc *HiveotSseServerConnection) Close() {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	if sc.IsConnected() {
		if sc.sseChan != nil {
			close(sc.sseChan)
		}
		sc.ConnectionBase.Disconnect()
	}
}

// Handle received notification message.
func (sc *HiveotSseServerConnection) onNotificationMessage(notif msg.NotificationMessage) {
	// TODO: is this handled here or does this use http-basic
}

func (sc *HiveotSseServerConnection) onResponseMessage(notif msg.ResponseMessage) {
	// TODO: is this handled here or does this use http-basic

}

// Handle received request messages.
//
// A response is only expected if the request is handled, otherwise nil is returned
// and a response is received asynchronously.
// In case of subscriptions, these are handled using the ConnectionBase.
// In case of invoke-action, the response is always an ActionStatus object.
//
// This returns one of 3 options:
// 1. on completion, return handled=true, an optional output
// 2. on error, return handled=true, output optional error details and error the error message
// 3. on async status, return handled=false, output optional, error nil
func (sc *HiveotSseServerConnection) onRequestMessage(
	req *msg.RequestMessage) (handled bool, output *msg.ResponseMessage, err error) {

	// handle subscriptions
	handled = true
	switch req.Operation {
	case wot.OpSubscribeEvent, wot.OpSubscribeAllEvents:
		sc.SubscribeEvent(req.ThingID, req.Name, req.CorrelationID)
	case wot.OpUnsubscribeEvent, wot.OpUnsubscribeAllEvents:
		sc.UnsubscribeEvent(req.ThingID, req.Name)
	case wot.OpObserveProperty, wot.OpObserveAllProperties:
		sc.ObserveProperty(req.ThingID, req.Name, req.CorrelationID)
	case wot.OpUnobserveProperty, wot.OpUnobserveAllProperties:
		sc.UnobserveProperty(req.ThingID, req.Name)
	default:
		handled = false
	}
	if handled {
		// subscription requests dont have output
		err = nil
		return handled, nil, nil
	}
	// note handled, pass it to the application
	hPtr := sc.appRequestHandlerPtr.Load()
	if hPtr == nil {
		// internal error
		err = fmt.Errorf("HiveotSseServerConnection:onRequestMessage: no request handler registered")
		return true, nil, err
	}

	// send the request and receive a response via the lambda
	err = (*hPtr)(req, func(resp *msg.ResponseMessage) error {
		// this only obtains a result for synchronous requests
		output = resp
		handled = true
		err = nil
		if resp.Error != nil {
			err = resp.Error.AsError()
			handled = true
		}
		return err
	})

	// responses are optional
	if !handled {
		// no response yet, return a 201
		err = nil
		output = nil
	}
	return handled, output, err
}

// SendNotification sends a notification to the client if subscribed.
func (sc *HiveotSseServerConnection) SendNotification(
	notif *msg.NotificationMessage) (err error) {

	clientID := sc.GetClientID()

	if sc.HasSubscription(notif) {
		// hiveotSSE sends messages as-is and does not use a message converter
		// msg, err := sc.messageConverter.EncodeNotification(notif)
		slog.Info("SendNotification (subscribed)",
			slog.String("clientID", clientID),
			slog.String("thingID", notif.ThingID),
			slog.String("op", notif.Operation),
			slog.String("name", notif.Name),
		)
		err = sc._send(msg.MessageTypeNotification, notif)
	} else {
		slog.Warn("Unknown notification: " + notif.Operation)
		//err = c._send(msg)
	}
	return err
}

// SendRequest sends a request message to an agent over SSE.
// If responseHandler is provided then the response is received via http using rnrChan.
func (sc *HiveotSseServerConnection) SendRequest(
	req *msg.RequestMessage, responseHandler msg.ResponseHandler) (err error) {

	// This sends the message as-is over SSE
	// The async response is expected over HTTP.
	if responseHandler == nil {
		err = sc._send(msg.MessageTypeRequest, req)
		return err
	}

	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	// the channel that will receives the result over http
	sc.rnrChan.WaitWithCallback(req.CorrelationID, responseHandler, sc.respTimeout)

	err = sc._send(msg.MessageTypeRequest, req)
	if err != nil {
		slog.Warn("SendRequest ->: error in sending request",
			"dThingID", req.ThingID,
			"name", req.Name,
			"correlationID", req.CorrelationID,
			"err", err.Error())
	}
	return err
}

// SendResponse send a response from server to client over SSE.
func (sc *HiveotSseServerConnection) SendResponse(resp *msg.ResponseMessage) error {
	// This simply sends the message as-is
	return sc._send(msg.MessageTypeResponse, resp)
}

// Serve serves SSE cm.
// This listens for outgoing requests on the given channel
// It ends when the client disconnects or the connection is closed with Close()
// Sse requests are refused if no valid session is found.
func (sc *HiveotSseServerConnection) Serve(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE response
	//w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate, max-age=0, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Content-Encoding", "none") //https://stackoverflow.com/questions/76375157/client-not-receiving-server-sent-events-from-express-js-server

	// establish a client event channel for sending messages back to the client
	sc.mux.Lock()
	sc.sseChan = make(chan SSEEvent, 1)
	sc.mux.Unlock()

	// _send a ping event as the go-sse client doesn't have a 'connected callback'
	pingEvent := SSEEvent{EventType: SSEPingEvent}
	sc.mux.Lock()
	sc.sseChan <- pingEvent
	sc.mux.Unlock()

	slog.Debug("SseConnection.Serve new SSE connection",
		slog.String("clientID", sc.GetClientID()),
		slog.String("connectionID", sc.ConnectionID),
		slog.String("protocol", r.Proto),
		slog.String("remoteAddr", sc.remoteAddr),
	)
	sendLoop := true

	// close the channel when the connection drops
	go func() {
		select {
		case <-r.Context().Done(): // remote client connection closed
			slog.Debug("SseConnection: Remote client disconnected (read context)")
			// close channel when no-one is writing
			// in the meantime keep reading to prevent deadlock
			sc.Disconnect()

		}
	}()

	// read the message channel for sending messages until it closes
	for sendLoop { // sseMsg := range sseChan {
		select {
		// keep reading to prevent blocking on channel on write
		case sseMsg, ok := <-sc.sseChan: // received event
			var err error

			if !ok { // channel was closed by session
				// avoid further writes
				sendLoop = false
				// ending the read loop and returning will close the connection
				break
			}
			slog.Debug("SseConnection: sending sse event to client",
				//slog.String("sessionID", c.sessionID),
				slog.String("clientID", sc.GetClientID()),
				slog.String("connectionID", sc.ConnectionID),
				slog.String("sse eventType", sseMsg.EventType),
			)
			var n int
			n, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n",
				sseMsg.EventType, sseMsg.Payload)
			//_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n",
			//	sseMsg.EventType, sseMsg.ID, sseMsg.Payload)
			if err != nil {
				// the connection might be closing.
				// don't exit the loop until the receive-channel is closed.
				// just keep processing the message until that happens
				// closed go channels panic when written to. So keep reading.
				slog.Error("SseConnection: Error writing SSE event",
					slog.String("Event", sseMsg.EventType),
					slog.String("SenderID", sc.GetClientID()),
					slog.Int("size", len(sseMsg.Payload)),
				)
			} else {
				slog.Debug("SseConnection: SSE write to client",
					slog.String("SenderID", sc.GetClientID()),
					slog.String("Event", sseMsg.EventType),
					slog.Int("N bytes", n))
			}
			w.(http.Flusher).Flush()
		}
	}
	//cs.DeleteSSEChan(sseChan)
	slog.Debug("SseConnection.Serve: sse connection closed",
		slog.String("remote", r.RemoteAddr),
		slog.String("clientID", sc.GetClientID()),
		slog.String("connectionID", sc.ConnectionID),
	)
}

// SetConnectHandler set the connection changed callback. Used by the connection manager.
func (sc *HiveotSseServerConnection) SetConnectHandler(cb transports.ConnectionHandler) {
	sc.mux.Lock()
	sc.connectionHandler = cb
	sc.mux.Unlock()
}

// SetNotificationHandler sets the handler for incoming notification messages from the
// http connection.
//
// Handlers of notifications must register a callback using SetNotificationHandler on the connection.
//
// Note on how this works: The global http server receives notifications as http requests.
// To make it look like the notification came from this connection it looks up the
// connection using the clientID and connectionID and passes the message to the
// registered notification handler on this connection.
//
// By default, the server registers itself as the notification handler when the connection
// is created. It is safe to set a different handler for applications that
// handle each connection separately, for example a server side consumer instance.
func (sc *HiveotSseServerConnection) SetNotificationHandler(cb msg.NotificationHandler) {
	if cb == nil {
		sc.notificationHandlerPtr.Store(nil)
	} else {
		sc.notificationHandlerPtr.Store(&cb)
	}
}

// SetRequestHandler sets the handler for incoming request messages from the
// http connection.
//
// The hiveot server design requires that the messages are coming from the connections.
// Handlers of requests must register a callback using SetRequestHandler on the connection.
//
// Note on how this works: The global http server receives http requests. To make
// it look like the request came from this connection it looks up the connection
// using the clientID and connectionID and passes the message to the registered
// request handler on this connection.
//
// By default, the server registers itself as the request handler when the connection
// is created. It is safe to set a different request handler for applications that
// handle each connection separately, for example an 'Agent' instance.
func (sc *HiveotSseServerConnection) SetRequestHandler(cb msg.RequestHandler) {
	if cb == nil {
		sc.appRequestHandlerPtr.Store(nil)
	} else {
		sc.appRequestHandlerPtr.Store(&cb)
	}
}

// SetResponseHandler sets the handler for incoming response messages from the
// http connection.
//
// The hiveot server design requires that the messages are coming from the connections.
// Handlers of responses must register a callback using SetResponseHandler on the connection.
//
// Note on how this works: The global http server receives responses as http requests.
// To make it look like the response came from this connection it looks up the
// connection using the clientID and connectionID and passes the message to the
// registered response handler on this connection.
//
// By default, the server registers itself as the response handler when the connection
// is created. It is safe to set a different response handler for applications that
// handle each connection separately, for example a server side consumer instance.
func (sc *HiveotSseServerConnection) SetResponseHandler(cb msg.ResponseHandler) {
	if cb == nil {
		sc.responseHandlerPtr.Store(nil)
	} else {
		sc.responseHandlerPtr.Store(&cb)
	}
}

// NewHiveotSseConnection creates a new SSE connection instance.
// This implements the IServerTransport interface.
//
// clientID is the authenticated ID of the client that just connected
// cid is the client's instance connectionID
// remoteAdd is the address used to connect.
// httpReq is the request that started the websocket connection
// rnrChan is the http server request&response channel where responses are passed.
func NewHiveotSseConnection(
	clientID string, cid string, remoteAddr string, httpReq *http.Request,
	rnrChan *transports.RnRChan) *HiveotSseServerConnection {

	c := &HiveotSseServerConnection{
		remoteAddr:   remoteAddr,
		httpReq:      httpReq,
		lastActivity: time.Now(),
		mux:          sync.RWMutex{},
		// observations:  connections.Subscriptions{},
		// subscriptions: connections.Subscriptions{},
		correlData: make(map[string]chan any),
		rnrChan:    rnrChan,
	}
	// c.isConnected.Store(true)
	c.Init(clientID, httpReq.URL.String(), cid)

	// interface check
	var _ transports.IServerConnection = c
	return c
}
