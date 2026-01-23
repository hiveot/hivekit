package clients

import (
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	httpbasicclient "github.com/hiveot/hivekit/go/modules/transports/httpbasic/client"
	ssescclient "github.com/hiveot/hivekit/go/modules/transports/ssesc/client"
	wssclient "github.com/hiveot/hivekit/go/modules/transports/wss/client"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/teris-io/shortid"
)

// Consumer is a module representing a WoT consumer.
// This implements the IHiveModule interface.
//
// The consumer is linked to a transport client from which it receives notification
// and through which it sends requests.
//
// There are 2 ways an application can receive incoming messages:
// 1. register handlers using SetAppNotificationHandler, SetAppRequestHandler, SetAppResponseHandler
// 2. override the HandleNotification, HandleRequest, HandleResponse methods
//
// TODO: To Sink or not to Sink?
// Should applications that use consumer provide a sink, or register handlers to receive messages?
// Does the use of a consumer imply this is the end of a the line for chaining messages?
//
// This is best used by embedding in the application and providing the application
// as the sink to a client or server connection.
// Alternatively, the application can override the HandleNotification|Request|Response methods
//
// Consumers can register callbacks for receiving events, updates of properties and changes in
// the connection.
//
// This implements the IHiveModule interface so it can be used as a sink for transports
// or other modules.
type Consumer struct {
	// This consumer is a sink for the connection
	// modules.HiveModuleBase
	appID string

	// application callback for reporting connection status change
	appConnectHandlerPtr atomic.Pointer[func(connected bool, err error, c transports.IConnection)]

	// application callback that handles asynchronous responses
	appResponseHandlerPtr atomic.Pointer[func(msg *msg.ResponseMessage) error]

	// application callback that handles notifications
	appNotificationHandlerPtr atomic.Pointer[func(msg *msg.NotificationMessage)]

	// The authenticated transport connection for delivering and receiving requests and responses
	cc transports.IConnection

	mux sync.RWMutex

	// The timeout to use when waiting for a response
	rpcTimeout time.Duration
}

// GetClientID returns the client's account ID
func (co *Consumer) GetClientID() string {
	return co.cc.GetClientID()
}

// GetModuleID returns the application
func (co *Consumer) GetModuleID() string {
	return co.appID
}

// GetConnection returns the underlying connection of this consumer
func (co *Consumer) GetConnection() transports.IConnection {
	return co.cc
}

// GetTM returns empty
func (co *Consumer) GetTM() string {
	return ""
}

// HandleNotification passes notifications to the registered application handler.
func (co *Consumer) HandleNotification(notif *msg.NotificationMessage) {

	hPtr := co.appNotificationHandlerPtr.Load()
	if hPtr == nil {
		if notif.Operation == wot.OpInvokeAction {
			// not everyone is interested in action progress updates
			slog.Info("HandleNotification: Action progress received. No handler registered",
				"operation", notif.Operation,
				"clientID", co.GetClientID(),
				"thingID", notif.ThingID,
				"name", notif.Name,
			)
		} else {
			// When subscribing to notifications, then a handler is expected
			slog.Error("HandleNotification: Notification received but no handler registered",
				"correlationID", notif.CorrelationID,
				"operation", notif.Operation,
				"clientID", co.GetClientID(),
				"thingID", notif.ThingID,
				"name", notif.Name,
			)
		}
		return
	}
	// pass the response to the registered handler
	slog.Info("HandleNotification",
		"operation", notif.Operation,
		"clientID", co.GetClientID(),
		"thingID", notif.ThingID,
		"name", notif.Name,
		"value", notif.ToString(50),
	)
	(*hPtr)(notif)
}

func (co *Consumer) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	return fmt.Errorf("Unexpected request op='%s', thingID='%s', name='%s', from '%s'",
		req.Operation, req.ThingID, req.Name, req.SenderID)
}

// HandleResponse passes a async responses to the registered app response handler.
// Used to pass a response from SendRequest when no replyTo is provided.
// This logs an error if no handler is set.
func (co *Consumer) HandleResponse(resp *msg.ResponseMessage) error {

	// handle the response as an async response with no wait handler registered
	hPtr := co.appResponseHandlerPtr.Load()
	if hPtr == nil {
		// at least one of the handlers should be registered
		slog.Error("Response received but no handler registered",
			"correlationID", resp.CorrelationID,
			"operation", resp.Operation,
			"clientID", co.GetClientID(),
			"thingID", resp.ThingID,
			"name", resp.Name,
		)
		err := fmt.Errorf("response received but no handler registered")
		return err
	}
	// pass the response to the registered handler
	slog.Info("onResponse (async)",
		"operation", resp.Operation,
		"clientID", co.GetClientID(),
		"thingID", resp.ThingID,
		"name", resp.Name,
		"value", resp.ToString(50),
	)
	return (*hPtr)(resp)
}

// InvokeAction invokes an action on a thing and wait for the response
// If the response type is known then provide it with output, otherwise use interface{}
func (co *Consumer) InvokeAction(
	thingID, name string, input any, output any) error {

	err := co.Rpc(wot.OpInvokeAction, thingID, name, input, output)
	// req := msg.NewRequestMessage(wot.OpInvokeAction, dThingID, name, input, "")
	// resp, err := co.SendRequest(req, true)

	// if err != nil {
	// 	return err
	// } else if resp.Error != nil {
	// 	return resp.Error.AsError()
	// }
	// err = resp.Decode(output)
	return err
}

// IsConnected returns true if the consumer has a connection
func (co *Consumer) IsConnected() bool {
	return co.cc.IsConnected()
}

// Logout requests invalidating all client sessions.
//func (co *WotClient) Logout() (err error) {
//
//	slog.Info("Logout",
//		slog.String("clientID", co.GetClientID()))
//
//	req := transports.NewRequestMessage(wot.HTOpLogout, "", "", nil, "")
//	_, err = co.SendRequest(req, true)
//	return err
//}

// ObserveProperty sends a request to observe one or all properties
//
//	thingID is empty for all things
//	name is empty for all properties of the selected things
func (co *Consumer) ObserveProperty(thingID string, name string) error {
	op := wot.OpObserveProperty
	if name == "" {
		op = wot.OpObserveAllProperties
	}

	err := co.Rpc(op, thingID, name, nil, "")
	return err
}

// connection status handler
func (co *Consumer) onConnect(connected bool, err error, c transports.IConnection) {
	hPtr := co.appConnectHandlerPtr.Load()
	if hPtr != nil {
		(*hPtr)(connected, err, c)
	}
}

// Ping the server and wait for a response.
// Intended to ensure the server is reachable.
func (co *Consumer) Ping() (err error) {
	var value any

	err = co.Rpc(wot.HTOpPing, "", "", nil, &value)
	if err != nil {
		return err
	}
	if value == nil {
		return errors.New("ping returned successfully but received no data")
	}
	return nil
}

// QueryAction obtains the status of an action
//
// Q: http-basic protocol returns an array per action in QueryAllActions but only
//
//	a single action in QueryAction. This is inconsistent.
//
// The underlying protocol binding constructs the ActionStatus from the
// protocol specific messages.
// The hiveot protocol passes this as-is as the output.
func (co *Consumer) QueryAction(thingID, name string) (
	value msg.ActionStatus, err error) {

	err = co.Rpc(wot.OpQueryAction, thingID, name, nil, &value)
	// if state is empty then this action has not run before
	if err == nil && value.State == "" {
		value.ThingID = thingID
		value.Name = name
	}
	return value, err
}

// QueryAllActions returns a map of action status for all actions of a thing.
//
// This returns a map of actionName and the last known action status.
//
// Q: http-basic protocol returns an array for each action. What is the use-case?
//
//	that can have multiple concurrent actions? An actuator can only move in
//	one direction at the same time.
//	Maybe the array only applies to stateless actions?
//
// This depends on the underlying protocol binding to construct appropriate
// ActionStatus message. All hiveot protocols include full information.
// WoT bindings might not include update timestamp and such.
func (co *Consumer) QueryAllActions(thingID string) (
	values map[string]msg.ActionStatus, err error) {

	err = co.Rpc(wot.OpQueryAllActions, thingID, "", nil, &values)
	return values, err
}

// ReadAllEvents sends a request to read all Thing event values from the hub.
//
// This returns a map of eventName and the last received event message.
//
// TODO: maybe better to send the last events on subscription...
//func (co *WotClient) ReadAllEvents(thingID string) (
//	values map[string]transports.ThingValue, err error) {
//
//	err = co.Rpc(wot.HTOpReadAllEvents, thingID, "", nil, &values)
//	return values, err
//}

// ReadAllProperties sends a request to read all Thing property values.
//
// This depends on the underlying protocol binding to construct appropriate
// ResponseMessages and include information such as Timestamp. All hiveot protocols
// include full information. WoT bindings might be more limited.
func (co *Consumer) ReadAllProperties(thingID string) (
	values map[string]msg.ThingValue, err error) {

	err = co.Rpc(wot.OpReadAllProperties, thingID, "", nil, &values)
	return values, err
}

// ReadAllTDs sends a request to read all TDs from an agent
// This returns an array of TDs in JSON format
// This is not a WoT operation (but maybe it should be)
//func (co *WotClient) ReadAllTDs() (tdJSONs []string, err error) {
//	err = co.Rpc(wot.HTOpReadAllTDs, "", "", nil, &tdJSONs)
//	return tdJSONs, err
//}

// ReadEvent sends a request to read a Thing event value.
//
// This returns a ResponseMessage containing the value as described in the TD
// event affordance schema.
//
// TODO: maybe better to send the last events on subscription...
//func (co *WotClient) ReadEvent(thingID, name string) (
//	value transports.ThingValue, err error) {
//
//	err = co.Rpc(wot.HTOpReadEvent, thingID, name, nil, &value)
//	return value, err
//}

// ReadProperty sends a request to read a Thing property value.
//
// This depends on the underlying protocol binding to construct appropriate
// ResponseMessages and include information such as Timestamp. All hiveot protocols
// include full information. WoT bindings might be too limited.
func (co *Consumer) ReadProperty(thingID, name string) (
	value msg.ThingValue, err error) {

	err = co.Rpc(wot.OpReadProperty, thingID, name, nil, &value)
	return value, err
}

// RetrieveThing sends a request to read the latest Thing TD
// This returns the TD in JSON format.
// This is not a WoT operation (but maybe it should be)
//func (co *WotClient) RetrieveThing(thingID string) (tdJSON string, err error) {
//	err = co.Rpc(wot.HTOpReadTD, thingID, "", nil, &tdJSON)
//	return tdJSON, err
//}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'SetBearerToken'
// This is specific to the Hiveot Hub.
//func (co *WotClient) RefreshToken(oldToken string) (newToken string, err error) {
//
//	// FIXME: what is the WoT standard for refreshing a token using http?
//	slog.Info("RefreshToken",
//		slog.String("clientID", co.GetClientID()))
//
//	req := transports.NewRequestMessage(wot.HTOpRefresh, "", "", oldToken, "")
//	resp, err := co.SendRequest(req, true)
//
//	// set the new token as the bearer token
//	if err == nil {
//		newToken = tputils.DecodeAsString(resp.Value, 0)
//	}
//	return newToken, err
//}

// Rpc sends a request message and waits for a response.
// This returns an error if the request fails or if the response contains an error
func (co *Consumer) Rpc(operation, thingID, name string, input any, output any) error {
	correlationID := shortid.MustGenerate()

	var resp *msg.ResponseMessage
	req := msg.NewRequestMessage(operation, thingID, name, input, correlationID)

	ar := utils.NewAsyncReceiver[*msg.ResponseMessage]()
	err := co.SendRequest(req, func(resp *msg.ResponseMessage) error {
		var err2 error
		slog.Info("Consumer RPC. Received response", "op", operation)
		if resp != nil {
			if resp.Error != nil {
				err2 = resp.Error.AsError()
			}
		}
		ar.SetResponse(resp, err2)
		return nil
	})
	if err == nil {
		resp, err = ar.WaitForResponse(co.rpcTimeout)
	}
	if err == nil && resp != nil {
		err = resp.Decode(output)
	}
	return err
}

// SendRequest sends an operation request and passes the response to the replyTo handler.
//
// If replyTo is nil then responses will go to the async response handler 'HandleResponse'.
//
// If the request has no correlation ID, one will be generated.
func (co *Consumer) SendRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	t0 := time.Now()
	slog.Info("SendRequest: ->",
		slog.String("op", req.Operation),
		slog.String("dThingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("correlationID", req.CorrelationID),
		slog.String("input", req.ToString(30)),
	)
	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	// if not waiting then return asap and pass the response to the async handler
	err = co.cc.SendRequest(req, func(resp *msg.ResponseMessage) error {
		var err2 error
		// intercept the response for logging and timing.
		t1 := time.Now()
		duration := t1.Sub(t0)

		errMsg := ""
		if resp.Error != nil {
			errMsg = resp.Error.String()
		}
		slog.Info("SendRequest: <-",
			slog.String("op", req.Operation),
			slog.Float64("duration msec", float64(duration.Microseconds())/1000),
			slog.String("correlationID", req.CorrelationID),
			slog.String("err", errMsg),
			slog.String("output", resp.ToString(30)),
		)
		if replyTo != nil {
			err2 = replyTo(resp)
		} else {
			err2 = co.HandleResponse(resp)
		}
		return err2
	})
	return err
}

// SetConnectHandler sets the connection callback for changes to this consumer connection
// Intended to notify the client that a reconnect or relogin is needed.
// Only a single handler is supported. This replaces the previously set callback.
func (co *Consumer) SetConnectHandler(
	cb func(connected bool, err error, c transports.IConnection)) {
	if cb == nil {
		co.appConnectHandlerPtr.Store(nil)
	} else {
		co.appConnectHandlerPtr.Store(&cb)
	}
}

// SetNotificationHandler sets the notification message callback for this consumer
// Only a single handler is supported. This replaces the previously set callback.
func (co *Consumer) SetNotificationHandler(cb func(msg *msg.NotificationMessage)) {
	if cb == nil {
		co.appNotificationHandlerPtr.Store(nil)
	} else {
		co.appNotificationHandlerPtr.Store(&cb)
	}
}

// SetResponseHandler set the handler that receives asynchronous responses
// Those are responses to requests that are not waited for using the baseRnR handler.
func (co *Consumer) SetResponseHandler(cb func(msg *msg.ResponseMessage) error) {
	if cb == nil {
		co.appResponseHandlerPtr.Store(nil)
	} else {
		co.appResponseHandlerPtr.Store(&cb)
	}
}

// Start using the consumer
func (co *Consumer) Start() error {
	return nil
}

// Stop the consumer module and closes the client connection.
func (co *Consumer) Stop() {
	if co.cc.IsConnected() {
		co.cc.Close()
		// the connect callback is still needed to notify the client of a disconnect
	}
}

// Subscribe to one or all events of a thing.
// name is the event to subscribe to or "" for all events
func (co *Consumer) Subscribe(thingID string, name string) error {
	op := wot.OpSubscribeEvent
	if name == "" {
		op = wot.OpSubscribeAllEvents
	}
	err := co.Rpc(op, thingID, name, nil, "")
	return err
}

// UnobserveProperty a previous observed property or all properties
func (co *Consumer) UnobserveProperty(thingID string, name string) error {
	op := wot.OpUnobserveProperty
	if name == "" {
		op = wot.OpUnobserveAllProperties
	}
	err := co.Rpc(op, thingID, name, nil, "")
	return err
}

// Unsubscribe is a helper for sending an unsubscribe request
func (co *Consumer) Unsubscribe(thingID string, name string) error {
	op := wot.OpUnsubscribeEvent
	if name == "" {
		op = wot.OpUnsubscribeAllEvents
	}
	err := co.Rpc(op, thingID, name, nil, "")
	return err
}

// WriteProperty is a helper to send a write property request
// Since writing properties can take some time on slow devices, the wait is optional.
func (co *Consumer) WriteProperty(thingID string, name string, input any, wait bool) (err error) {
	correlationID := shortid.MustGenerate()
	if wait {
		err = co.Rpc(wot.OpWriteProperty, thingID, name, input, correlationID)
	} else {
		req := msg.NewRequestMessage(wot.OpWriteProperty, thingID, name, input, correlationID)
		err = co.SendRequest(req, func(resp *msg.ResponseMessage) error {
			// just ignore the result
			return nil
		})
	}
	return err
}

// NewClient returns a new client instance ready to connect
func NewClient(serverURL string, caCert *x509.Certificate, timeout time.Duration) (cl transports.IConnection, err error) {
	parts, err := url.Parse(serverURL)
	scheme := strings.ToLower(parts.Scheme)
	var sink modules.IHiveModule

	switch scheme {
	case transports.ProtocolTypeHiveotSSE: // "sse"
		cl = ssescclient.NewSseScClient(serverURL, caCert, sink, timeout)

	case transports.ProtocolTypeWotWSS: // "wss"
		cl = wssclient.NewWotWssClient(serverURL, caCert, sink, timeout)

	case transports.ProtocolTypeHTTPBasic: // "https"
		caCert := caCert
		cl = httpbasicclient.NewHttpBasicClient(
			serverURL, caCert, sink, nil, timeout)

	//case transports.ProtocolTypeWotMQTTWSS:
	//	fullURL = testServerMqttWssURL

	default:
		err = fmt.Errorf("NewClient. Unknown protocol '%s'", scheme)
	}
	return cl, err
}

// NewConsumer returns a new instance of the WoT consumer for use with the given
// connection. This consumer takes possession of the provided client connection
// by registering connection callbacks.
//
// This provides the API for common WoT operations such as invoking actions and
// supports RPC calls by waiting for a response.
//
// Use SetNotificationHandler to set the callback to receive async notifications.
// Use SetResponseHandler to set the callback to receive async responses.
// Use SetConnectHandler to set the callback to be notified of connection changes.
//
//	appID the ID of this application
//	cc the client connection to use for sending requests and receiving responses.
//	timeout of the rpc connections or 0 for default (3 sec)
func NewConsumer(appID string, cc transports.IConnection, rpcTimeout time.Duration) *Consumer {
	if rpcTimeout == 0 {
		rpcTimeout = transports.DefaultRpcTimeout
	}
	consumer := &Consumer{
		cc:    cc,
		appID: appID,
		// rnrChan:    NewRnRChan(),
		rpcTimeout: rpcTimeout,
	}
	// consumer.Init(moduleID, nil)
	consumer.SetNotificationHandler(nil)
	consumer.SetConnectHandler(nil)
	consumer.SetResponseHandler(nil)
	// set the connection callbacks to this consumer

	// This consumer is the sink for the transport client, all messages are forwarded here.
	// Consumers also use it to send messages.
	cc.SetSink(consumer)
	cc.SetConnectHandler(consumer.onConnect)
	return consumer
}

// NewConsumerConnection creates a client connection and returns a new instance of
// a WoT consumer.
//
// This provides the API for common WoT operations such as invoking actions and
// supports RPC calls by waiting for a response.
//
// Use SetNotificationHandler to set the callback to receive async notifications.
// Use SetResponseHandler to set the callback to receive async responses.
// Use SetConnectHandler to set the callback to be notified of connection changes.
//
//	appID the ID of this application
//	serverURL is the full URL identifying the protocol using its schema (wss, sse, https, mqtt)
//	caCert is the server's CA certificate or nil to disable this important check.
//	rpcTimeout of the rpc connections or 0 for default (3 sec)
//
// This returns the consumer and the client connection. The caller still needs to call
// one of the ConnectWith... methods to provide the credentials.
func NewConsumerConnection(
	appID string, serverURL string, caCert *x509.Certificate, rpcTimeout time.Duration) (
	*Consumer, transports.IConnection, error) {

	cc, err := NewClient(serverURL, caCert, rpcTimeout)
	if err != nil {
		return nil, nil, err
	}
	consumer := NewConsumer(appID, cc, rpcTimeout)
	return consumer, cc, nil
}
