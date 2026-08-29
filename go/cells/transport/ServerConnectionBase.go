package transport

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/teris-io/shortid"
)

// ServerConnectionBase is a generic base type for implementing server side
// transport connections.
//
// Use of this is totally optional and might not apply to all transport.
//
// Features:
//  1. Handle received messages
//     1a. ping request
//     1b. subscription requests (OnRequest)
//  2. Handle received response and pass it to the RnR handler (OnResponse)
//  3. Handle received notification (OnNotification)
//
// 4a. SendRequest to remote - needs encoder and sendRaw provided in Init
// 4b. SendResponse to remote - needs encoder and sendRaw provided in Init
// 4c. SendNotification to remote - needs encoder and sendRaw provided in Init
// 4d. SendNotification filters on subscriptions
//
// This implements the IConnection interface.
type ServerConnectionBase struct {
	// Authenticated ID of the remote client
	ClientID string

	// The connection identification
	ConnectionID string

	// SendNotificationHandler msg.NotificationHandler
	// connections clients are asynchronous
	// SendRequestHandler func(req *msg.RequestMessage) (err error)
	//
	// SendResponseHandler msg.ResponseHandler
	isConnected atomic.Bool

	// messageEncoder for request/response messages
	messageEncoder IMessageEncoder

	// notifForwarder used by OnMessage to forward the decoded notifications to the server sink
	notifForwarder msg.NotificationHandler

	// reqForwarder used by OnRemoteMessage to forwards the decoded requests to the server sink
	reqForwarder msg.RequestHandler

	// property observations made by the client
	observations Subscriptions

	// request-response channel
	// Used by SendRequest to wait and pass a response to the sender replyTo
	rnrChan *msg.RnRChan

	// send encoded messages to the remote client
	sendRaw func(msgType string, raw []byte) error

	// Remote address of the connection
	remoteAddr string

	// how long to wait for a response after sending a request
	respTimeout time.Duration

	// event subscription made by the client
	subscriptions Subscriptions

	// Mux to update subscriptions, connection status
	Mux sync.RWMutex
}

func (scb *ServerConnectionBase) Disconnect() {
	scb.isConnected.Store(false)
}

func (scb *ServerConnectionBase) GetClientID() string {
	return scb.ClientID
}

func (scb *ServerConnectionBase) GetConnectionID() string {
	return scb.ConnectionID
}

// // GetConnectionStatus returns the current connection status
func (sc *ServerConnectionBase) GetConnectionStatus() api.ConnectionStatus {
	if sc.isConnected.Load() {
		return api.StatusConnected
	}
	return api.StatusLost
}

// HasSubscription returns true if this connection has subscribed to the given
// event notification or observing property changes.
func (scb *ServerConnectionBase) HasSubscription(notif *msg.NotificationMessage) bool {
	switch notif.AffordanceType {

	case msg.AffordanceTypeEvent:
		correlationID := scb.subscriptions.GetSubscription(notif.ThingID, notif.Name)
		if correlationID != "" {
			slog.Info("HasSubscription (event subscription) -> true",
				slog.String("clientID", scb.ClientID),
				slog.String("thingID", notif.ThingID),
				slog.String("event name", notif.Name),
			)
			return true
		}
	case msg.AffordanceTypeProperty:
		correlationID := scb.observations.GetSubscription(notif.ThingID, notif.Name)
		if correlationID != "" {
			slog.Info("HasSubscription (observed property(ies)) -> true",
				slog.String("clientID", scb.ClientID),
				slog.String("thingID", notif.ThingID),
				slog.String("name", notif.Name),
			)
			return true
		}
	case msg.AffordanceTypeAction:
		// action progress update, for original sender only
		slog.Info("HasSubscription (action status) -> true",
			slog.String("clientID", scb.ClientID),
			slog.String("thingID", notif.ThingID),
			slog.String("name", notif.Name),
		)
		return true
	default:
		slog.Warn("HasSubscription: Unknown affordance: " + string(notif.AffordanceType))
	}
	return false
}

func (scb *ServerConnectionBase) IsConnected() bool {
	return scb.isConnected.Load()
}

func (scb *ServerConnectionBase) ObserveProperty(dThingID, name string, corrID string) {
	scb.observations.Subscribe(dThingID, name, corrID)
}

// OnNotification receives a notification from remote client (producer).
// pass it on to the registered (upstream) notification handler.
//
// remote producer [notification] -> client<=>server -> _onNotification
func (sc *ServerConnectionBase) OnNotification(
	notif *msg.NotificationMessage, forwardTo msg.NotificationHandler) {

	// sender is identified by the server, not the client
	notif.SenderID = sc.GetClientID()

	// we might add some counters or handle special notifications here in the future.
	forwardTo(notif)
}

// OnRemoteMessage handles an incoming message from remote client.
// The message is converted into a request, response or notification
// by the provided encoder and and passed on to the registered handler.
func (sc *ServerConnectionBase) OnRemoteMessage(raw []byte) {

	var err error
	if raw == nil {
		slog.Error("OnServerMessage: raw data is nil")
	}
	senderID := sc.GetClientID()
	msgType := sc.messageEncoder.DetermineMessageType(raw)

	switch msgType {
	case msg.MessageTypeNotification:
		var notif *msg.NotificationMessage
		notif, err = sc.messageEncoder.DecodeNotification(senderID, raw)
		if err == nil {
			sc.OnNotification(notif, sc.notifForwarder)
			return
		}
	case msg.MessageTypeRequest:
		var req *msg.RequestMessage
		req, err := sc.messageEncoder.DecodeRequest(senderID, raw)
		if err == nil {
			req.SenderID = sc.GetClientID()
			sc.OnRequest(req, sc.reqForwarder)
			return
		}
	case msg.MessageTypeResponse:
		var resp *msg.ResponseMessage
		resp, err := sc.messageEncoder.DecodeResponse(senderID, raw)
		if err == nil {
			resp.SenderID = sc.GetClientID()
			sc.OnResponse(resp)
			return
		}
	default:
		err = fmt.Errorf("Unknown message type '%s'", msgType)
	}
	if err != nil {
		slog.Warn("_onGrpcServerMessage: Failed to decode message", "msgType", msgType, "err", err.Error())
	}
}

// OnRequest is passed the request message sent by the remote client.
//
// This handles ping and subscription requests for the connection.
//
// All other requests are passed to the sink using the 'emitRequest' handler.
// The reply is send back as a response message the sender.
func (scb *ServerConnectionBase) OnRequest(
	req *msg.RequestMessage, emitRequest msg.RequestHandler) error {

	var resp *msg.ResponseMessage
	var err error

	// sender is identified by the server, not the client
	req.SenderID = scb.GetClientID()
	if req.Operation == "" {
		err = fmt.Errorf("OnRequest: no operation ")
		slog.Error(err.Error())
		return err
	}

	slog.Info("OnRequest",
		slog.String("senderID", scb.ClientID),
		slog.String("op", req.Operation),
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("correlationID", req.CorrelationID))

	switch req.Operation {
	case td.HTOpPing:
		resp = req.CreateResponse("pong", nil)

	case td.OpSubscribeEvent, td.OpSubscribeAllEvents:
		scb.SubscribeEvent(req.ThingID, req.Name, req.CorrelationID)
		resp = req.CreateResponse(nil, nil)

	case td.OpUnsubscribeEvent, td.OpUnsubscribeAllEvents:
		scb.UnsubscribeEvent(req.ThingID, req.Name)
		resp = req.CreateResponse(nil, nil)

	case td.OpObserveProperty, td.OpObserveAllProperties:
		scb.ObserveProperty(req.ThingID, req.Name, req.CorrelationID)
		resp = req.CreateResponse(nil, nil)

	case td.OpUnobserveProperty, td.OpUnobserveAllProperties:
		scb.UnobserveProperty(req.ThingID, req.Name)
		resp = req.CreateResponse(nil, nil)
	default:
		// this is not a subscription to notifications so pass it to the server sink.
		err = emitRequest(req, func(reply *msg.ResponseMessage) error {
			// the callback is async so handle it separately
			if reply != nil {
				return scb.SendResponse(reply)
			} else {
				// not having a reply handler is an error
				err := fmt.Errorf("onRequest: Sink response callback without response")
				slog.Error(err.Error())
				return err
			}
		})
		// if handling the request failed, return an error response
		if err != nil {
			resp = req.CreateErrorResponse(err)
		}
	}
	if resp != nil {
		err = scb.SendResponse(resp)
	}
	if err != nil {
		slog.Warn("Error handling request message", "err", err.Error())
	}
	return err
}

// OnResponse is passed the received response message for passing it to
// the waiting handler.
//
// This sets the senderID to that of the connection and passes the response
// to the RnR response handler to serve the request handler that is waiting
// for a response.
// See also RnRChan.HandleResponse that does the heavy lifting.
// This returns an error if RnR is not expecting a response with the correlationID
func (sc *ServerConnectionBase) OnResponse(resp *msg.ResponseMessage) (err error) {
	// sender is identified by the server, not the client
	resp.SenderID = sc.GetClientID()

	slog.Info("OnResponse (from Thing)",
		slog.String("senderID", sc.ClientID),
		slog.String("op", resp.Operation),
		slog.String("thingID", resp.ThingID),
		slog.String("name", resp.Name),
		slog.String("correlationID", resp.CorrelationID))

	// this responsehandler points to the rnrChannel that matches the correlationID to the replyTo handler
	handled := sc.rnrChan.HandleResponse(resp, sc.respTimeout)
	if !handled {
		err = fmt.Errorf("OnResponse: Response from thingID/name '%s/%s' is lost. CorrelationID '%s' is not known",
			resp.ThingID, resp.Name, resp.CorrelationID)
	}
	return err
}

// SendNotification encodes the notifications and passes it to sendRaw
func (sc *ServerConnectionBase) SendNotification(notif *msg.NotificationMessage) {

	if sc.HasSubscription(notif) {
		slog.Info("SendNotification",
			slog.String("cid", sc.GetConnectionID()),
			slog.String("senderID", notif.SenderID),
			slog.String("clientID", sc.GetClientID()),
			slog.String("thingID", notif.ThingID),
			slog.String("affordance", string(notif.AffordanceType)),
			slog.String("name", notif.Name),
		)
		raw, err := sc.messageEncoder.EncodeNotification(notif)
		if err == nil {
			err = sc.sendRaw(msg.MessageTypeNotification, raw)
		}
		if err != nil {
			// maybe the connection dropped. It should have been removed though so something went wrong.
			slog.Warn("SendNotification: Unable to send the notification to the client",
				"clientID", sc.ClientID,
				"err", err.Error())
		}
	}
}

// SendRequest sends the request to the client (Thing).
//
// This accepts a response handler through which the response is received. If not
// provided then the response will be forwarded to the cell's sink.
//
// Intended to be used by gateways that forward requests from consumers to Things,
// where the Thing has connected to the gateway using (connection reversal) and the
// gateway forwards on behalf of the consumer.
//
// When a response is received it is passed to the replyTo handler.
//
// If this returns an error then no request was sent.
func (sc *ServerConnectionBase) SendRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	raw, err := sc.messageEncoder.EncodeRequest(req)
	if err != nil {
		return err
	}
	// without a replyTo simply send the request
	if replyTo == nil {
		err = sc.sendRaw(msg.MessageTypeRequest, raw)
		return err
	}

	// with a replyTo, send the response async to the replyTo handler
	// catch the response in a channel linked by correlation-id
	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	// the websocket connection response handlers will convert the message and pass it to the RNR channels
	sc.rnrChan.WaitWithCallback(req.CorrelationID, sc.respTimeout, replyTo)

	// now the RRN channel is ready, send the request message
	err = sc.sendRaw(msg.MessageTypeRequest, raw)
	return err
}

// SendResponse sends a response to the remote client.
// If this returns an error then no response was sent.
func (sc *ServerConnectionBase) SendResponse(resp *msg.ResponseMessage) (err error) {

	slog.Info("SendResponse (server->client)",
		slog.String("clientID", sc.ClientID),
		slog.String("correlationID", resp.CorrelationID),
		slog.String("operation", resp.Operation),
		slog.String("name", resp.Name),
		slog.String("status", resp.Status),
		slog.String("type", resp.MessageType),
		slog.String("senderID", resp.SenderID),
	)

	raw, _ := sc.messageEncoder.EncodeResponse(resp)
	err = sc.sendRaw(msg.MessageTypeResponse, raw)
	return err
}

// SetTimeout set the timeout sending requests
func (sc *ServerConnectionBase) SetTimeout(timeout time.Duration) {
	sc.respTimeout = timeout
}

// Subscribe to an event.
func (scb *ServerConnectionBase) SubscribeEvent(dThingID, name string, corrID string) {
	scb.subscriptions.Subscribe(dThingID, name, corrID)
}
func (scb *ServerConnectionBase) UnsubscribeEvent(dThingID, name string) {
	scb.subscriptions.Unsubscribe(dThingID, name)
}
func (scb *ServerConnectionBase) UnobserveProperty(dThingID, name string) {
	scb.observations.Unsubscribe(dThingID, name)
}

// NewServerConnectionBase creates a server connection base that implements most of the
// boilerplate of converting and handling messages and subscriptions.
//
// To use the SendReq/Notif/Resp messages provide the encoder and sendRaw methods.
// reqForwarder and notifForwarder are used by OnRemoteMessage to pass to OnRequest and OnNotification.
// If OnRemoteMessage is not used then these can be nil.
//
//	clientID of the remote client
//	remoteAddr is the remote client's endpoint address
//	cid is the connection ID to differentiate multiple connections from this client
//	encoder is used for encoding and decoding messages. nil defaults to json encoding of RRN messages
//	sendRaw is the underlying transport sending encoded messages
//	reqForwarder is the callback for forwarding incoming request messages from the remote client
//	notifForwarder is the callback for forwarding incoming notification messages from the remote client
func NewServerConnectionBase(
	clientID, remoteAddr, cid string,
	encoder IMessageEncoder,
	sendRaw func(msgType string, raw []byte) error,
	reqForwarder msg.RequestHandler,
	notifForwarder msg.NotificationHandler,
) *ServerConnectionBase {

	if encoder == nil {
		encoder = NewRRNJsonEncoder()
	}
	scb := &ServerConnectionBase{
		ClientID:       clientID,
		ConnectionID:   cid,
		remoteAddr:     remoteAddr,
		messageEncoder: encoder,
		reqForwarder:   reqForwarder,
		notifForwarder: notifForwarder,
		sendRaw:        sendRaw,
		rnrChan:        msg.NewRnRChan(),
	}
	scb.isConnected.Store(true)
	return scb
}
