package module

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/ssesc"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
)

type SSEEvent struct {
	EventType string // type of message, e.g. event, action or other
	Payload   string // message content
}

// HiveotSseServerConnection handles the SSE connection by remote client
//
// The Sse-sc (sse single connection) protocol binding uses a 'hiveot' message
// envelope for sending messages between server and consumer.
//
// The sse server connection is a 1-way connection intended for sending messages
// to a client that connects over SSE. The client will use http to send messages
// to the server.
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

	sseChan chan SSEEvent

	// the Request-and-Response helper that links responses from http
	// with requests send over sse.
	// This instance is owned by the http server which passes responses to it.
	rnrChan *msg.RnRChan
	// timeout to observe when waiting for responses
	respTimeout time.Duration
}

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
	} else {
		slog.Error("HiveotSseServerConnection unable to send message. Connection lost.",
			"msgType", msgType, "clientID", sc.ClientID)
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
// func (sc *HiveotSseServerConnection) onNotificationMessage(notif msg.NotificationMessage) {
// 	// TODO: is this handled here or does this use http-basic
// }

// func (sc *HiveotSseServerConnection) onResponseMessage(notif msg.ResponseMessage) {
// 	// TODO: is this handled here or does this use http-basic

// }

// onRequestMessage handles (un)subscribe and (un)observe requests.
// This sends a response to the client, confirming the subscription.
//
// Everything else returns with handled false.
func (sc *HiveotSseServerConnection) onRequestMessage(
	req *msg.RequestMessage) (handled bool, err error) {

	// handle subscriptions using connection base
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
		// confirm
		resp := req.CreateResponse(nil, nil)
		err = sc.SendResponse(resp)
	}
	return handled, err
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
		// ignore the notification
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

	// The module provided the RnR channel handling.
	// The response to this request will be received over HTTP via the module routes.
	// Once received, the response handler  it will pass it to the RnR channel which
	// in turn invokes this responseHandler callback.
	sc.rnrChan.WaitWithCallback(req.CorrelationID, responseHandler)

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
	pingEvent := SSEEvent{EventType: ssesc.SSEPingEvent}
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
			slog.Info("SseConnection: sending sse event to client",
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

// // SetConnectHandler set the connection changed callback. Used by the connection manager.
// func (sc *HiveotSseServerConnection) SetConnectHandler(cb transports.ConnectionHandler) {
// 	sc.mux.Lock()
// 	sc.connectionHandler = cb
// 	sc.mux.Unlock()
// }

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
	rnrChan *msg.RnRChan) *HiveotSseServerConnection {

	c := &HiveotSseServerConnection{
		remoteAddr:   remoteAddr,
		httpReq:      httpReq,
		lastActivity: time.Now(),
		mux:          sync.RWMutex{},

		rnrChan:     rnrChan,
		respTimeout: transports.DefaultRpcTimeout,
	}
	c.Init(clientID, httpReq.URL.String(), cid)

	// interface check
	var _ transports.IServerConnection = c
	return c
}
