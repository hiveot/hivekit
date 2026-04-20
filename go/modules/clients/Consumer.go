package clients

import (
	"crypto/x509"
	"errors"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	factory "github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/teris-io/shortid"
)

const ConsumerModuleType = "consumer"

// Consumer is a module representing a WoT consumer.
//
// This implements the IHiveModule interface and a number of convenience functions to
// construct requests for subscribing to events and properties, reading properties, etc.
//
// Use of this is optional as clients also just use HiveModuleBase and use the Rpc() method.
//
// Usage:
//
//	This module can be used as a base for service clients that like to use the
//	ready-to-use API for sending requests and querying properties.
//
//	While it uses a clientID, this ID is merely for testing convenience when no transport
//	or a direct transport is used. Normally the transport inserts the authenticated clientID
//	as the sender.
//
//	To use this consumer it needs to be linked to a transport client module in order to deliver requests
//	and receive notifications using one of the available transport protocols.
//
//	SetRequestSink(transportclient.Handlerequest) to set the transport for delivering requests.
//	SetNotificationSink(consumer.HandleNotification) to receive notifications from the client
//
// This implements the IHiveModule interface so it can be used as a sink for transports
// or other modules.
type Consumer struct {
	// This consumer is a sink for the connection
	modules.HiveModuleBase

	// clientID is the clientID this consumer identifies as
	clientID string

	// The sink that will forward the requests and respond with notifications.
	// sink modules.IHiveModule

	mux sync.RWMutex

	// The timeout to use when waiting for a response
	rpcTimeout time.Duration
}

// GetClientID returns the client's account ID
func (co *Consumer) GetClientID() string {
	return co.clientID
}

// InvokeAction invokes an action on a thing and wait for the response
// If the response type is known then provide it with output, otherwise use interface{}
func (co *Consumer) InvokeAction(
	thingID, name string, input any, output any) error {

	err := co.Rpc(co.clientID, td.OpInvokeAction, thingID, name, input, output)
	return err
}

// ObserveProperty sends a request to observe one or all properties
//
//	thingID is empty for all things
//	name is empty for all properties of the selected things
func (co *Consumer) ObserveProperty(thingID string, name string) error {
	op := td.OpObserveProperty
	if name == "" {
		op = td.OpObserveAllProperties
	}

	err := co.Rpc(co.clientID, op, thingID, name, nil, "")
	return err
}

// Ping the server and wait for a response.
// Intended to ensure the server is reachable.
func (co *Consumer) Ping() (err error) {
	var value any

	err = co.Rpc(co.clientID, td.HTOpPing, "", "", nil, &value)
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
	value msg.ResponseMessage, err error) {

	err = co.Rpc(co.clientID, td.OpQueryAction, thingID, name, nil, &value)
	// if state is empty then this action has not run before
	if err == nil && value.Status == "" {
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
	values map[string]msg.ResponseMessage, err error) {

	err = co.Rpc(co.clientID, td.OpQueryAllActions, thingID, "", nil, &values)
	return values, err
}

// ReadAllEvents sends a request to read all Thing event values from the hub.
//
// This returns a map of eventName and the last sent notification message.
func (co *Consumer) ReadAllEvents(thingID string) (
	values map[string]*msg.NotificationMessage, err error) {

	err = co.Rpc(co.clientID, td.HTOpReadAllEvents, thingID, "", nil, &values)
	return values, err
}

// ReadAllProperties sends a request to read all Thing property values.
//
// This returns a map of property name-value pairs as described in the TD.
func (co *Consumer) ReadAllProperties(thingID string) (
	values map[string]any, err error) {

	err = co.Rpc(co.clientID, td.OpReadAllProperties, thingID, "", nil, &values)
	return values, err
}

// ReadAllTDs sends a request to read all TDs from an agent
// This returns an array of TDs in JSON format
// This is not a WoT operation (but maybe it should be)
//func (co *WotClient) ReadAllTDs() (tdJSONs []string, err error) {
//	err = co.Rpc(td.HTOpReadAllTDs, "", "", nil, &tdJSONs)
//	return tdJSONs, err
//}

// ReadEvent sends a request to read the last event message sent by a Thing.
//
// This returns the NotificationMessage that was last sent, containing the timestamp
// and value as described in the event affordance.
func (co *Consumer) ReadEvent(thingID, name string) (value *msg.NotificationMessage, err error) {

	err = co.Rpc(co.clientID, td.HTOpReadEvent, thingID, name, nil, &value)
	return value, err
}

// ReadProperty sends a request to read the current value of a Thing property.
//
// This decodes the value into the provided type
func (co *Consumer) ReadProperty(thingID, name string, value any) (err error) {

	err = co.Rpc(co.clientID, td.OpReadProperty, thingID, name, nil, value)
	return err
}

// ReadPropertyAs sends a request to read the current value of a Thing property.
//
// This converts the property value to the given type or returns an error
func (co *Consumer) ReadPropertyAs(thingID, name string, prop any) (err error) {

	err = co.Rpc(co.clientID, td.OpReadProperty, thingID, name, nil, prop)
	return err
}

// RetrieveThing sends a request to read the latest Thing TD
// This returns the TD in JSON format.
// This is not a WoT operation (but maybe it should be)
//func (co *WotClient) RetrieveThing(thingID string) (tdJSON string, err error) {
//	err = co.Rpc(td.HTOpReadTD, thingID, "", nil, &tdJSON)
//	return tdJSON, err
//}

// // Rpc sends a request message and waits for a response.
// // This returns an error if the request fails or if the response contains an error
// func (co *Consumer) Rpc(operation, thingID, name string, input any, output any) error {
// 	correlationID := shortid.MustGenerate()

// 	var resp *msg.ResponseMessage
// 	req := msg.NewRequestMessage(operation, thingID, name, input, correlationID)

// 	resp, err := co.ForwardRequestWait(req)

// 	// ar := utils.NewAsyncReceiver[*msg.ResponseMessage]()
// 	// err := co.ForwardRequest(req, func(resp *msg.ResponseMessage) error {
// 	// 	slog.Info("Consumer RPC. Received response", "op", operation)
// 	// 	ar.SetResponse(resp)
// 	// 	return nil
// 	// })
// 	// if err == nil {
// 	// resp, err = ar.WaitForResponse(co.rpcTimeout)
// 	// }
// 	if err == nil && resp != nil {
// 		err = resp.Decode(output)
// 	}
// 	return err
// }

// ForwardRequest sends an operation request and passes the response to the replyTo handler.
//
// If replyTo is nil then responses are ignored.
//
// // If the request has no correlation ID, one will be generated.
// func (co *Consumer) SendRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

// 	t0 := time.Now()
// 	slog.Info("SendRequest: ->",
// 		slog.String("op", req.Operation),
// 		slog.String("dThingID", req.ThingID),
// 		slog.String("name", req.Name),
// 		slog.String("correlationID", req.CorrelationID),
// 		slog.String("input", req.ToString(30)),
// 	)
// 	// if req.CorrelationID == "" {
// 	// req.CorrelationID = shortid.MustGenerate()
// 	// }
// 	// if not waiting then return asap and pass the response to the async handler
// 	err = co.ForwardRequest(req, func(resp *msg.ResponseMessage) error {
// 		var err2 error
// 		// intercept the response for logging and timing.
// 		t1 := time.Now()
// 		duration := t1.Sub(t0)

// 		errMsg := ""
// 		if resp.Error != nil {
// 			errMsg = resp.Error.String()
// 		}
// 		slog.Info("SendRequest: <-",
// 			slog.String("op", req.Operation),
// 			slog.Float64("duration-msec", float64(duration.Microseconds())/1000),
// 			slog.String("correlationID", req.CorrelationID),
// 			slog.String("err", errMsg),
// 			slog.String("output", resp.ToString(30)),
// 		)
// 		if replyTo != nil {
// 			err2 = replyTo(resp)
// 		} else {
// 			slog.Info("SendRequest: no response handler provided")
// 		}
// 		return err2
// 	})
// 	return err
// }

// SetTimeout sets the RPC request handling timeout
func (co *Consumer) SetTimeout(timeout time.Duration) {
	co.rpcTimeout = timeout
}

// Subscribe to one or all events of a thing.
// name is the event to subscribe to or "" for all events
func (co *Consumer) Subscribe(thingID string, name string) error {
	op := td.OpSubscribeEvent
	if name == "" {
		op = td.OpSubscribeAllEvents
	}
	err := co.Rpc(co.clientID, op, thingID, name, nil, "")
	return err
}

// UnobserveProperty a previous observed property or all properties
func (co *Consumer) UnobserveProperty(thingID string, name string) error {
	op := td.OpUnobserveProperty
	if name == "" {
		op = td.OpUnobserveAllProperties
	}
	err := co.Rpc(co.clientID, op, thingID, name, nil, "")
	return err
}

// Unsubscribe is a helper for sending an unsubscribe request
func (co *Consumer) Unsubscribe(thingID string, name string) error {
	op := td.OpUnsubscribeEvent
	if name == "" {
		op = td.OpUnsubscribeAllEvents
	}
	err := co.Rpc(co.clientID, op, thingID, name, nil, "")
	return err
}

// WriteProperty is a helper to send a write property request
// Since writing properties can take some time on slow devices, the wait is optional.
func (co *Consumer) WriteProperty(thingID string, name string, input any, wait bool) (err error) {
	correlationID := shortid.MustGenerate()
	if wait {
		err = co.Rpc(co.clientID, td.OpWriteProperty, thingID, name, input, correlationID)
	} else {
		req := msg.NewRequestMessage(td.OpWriteProperty, thingID, name, input, correlationID)
		err = co.ForwardRequest(req, func(resp *msg.ResponseMessage) error {
			// just ignore the result
			return nil
		})
	}
	return err
}

// NewConsumer returns a new instance of the WoT consumer for use with the given
// connection.
//
// This provides the API for common WoT operations such as invoking actions and
// supports RPC calls by waiting for a response.
//
// Use SetSink to set the module that will handle requests and return notifications.
// Use SetTimeout to change the default timeout of RPC requests. (default: 3 sec)
//
//	appID the ID of this application
func NewConsumer(appID string) *Consumer {
	consumer := &Consumer{
		clientID:   appID,
		rpcTimeout: msg.DefaultRnRTimeout,
	}

	return consumer
}

// Factory for creating a consumer module using the factory environment
func NewConsumerFactory(f factory.IModuleFactory) modules.IHiveModule {
	appID := f.GetEnvironment().AppID
	c := NewConsumer(appID)
	return c
}

// NewConsumerConnection creates a client connection and returns a new instance of
// a WoT consumer and its connection.
//
// This provides the API for common WoT operations such as invoking actions and
// supports RPC calls by waiting for a response.
//
// Use SetNotificationHandler to set the callback to receive async notifications.
// Use SetResponseHandler to set the callback to receive async responses.
// Use SetConnectHandler to set the callback to be notified of connection changes.
//
//	clientID the ID of this application
//	serverURL is the full URL identifying the protocol using its schema (wss, sse, https, mqtt)
//	caCert is the server's CA certificate or nil to disable this important check.
//	rpcTimeout of the rpc connections or 0 for default (3 sec)
//
// This returns the consumer and the client connection.
// The caller still needs to call one of the ConnectWith... methods to provide the credentials.
// The caller must call client connection Stop or Close when done. The consumer cant do it.
func NewConsumerConnection(
	appID string, protocolType string, serverURL string,
	caCert *x509.Certificate, rpcTimeout time.Duration) (
	*Consumer, transports.ITransportClient, error) {

	cc, err := NewTransportClient(protocolType, serverURL, caCert, nil)
	cc.SetTimeout(rpcTimeout)
	if err != nil {
		return nil, nil, err
	}
	// set the connection as the sink that handles requests and publishes notifications
	consumer := NewConsumer(appID)
	consumer.SetRequestSink(cc.HandleRequest)
	cc.SetNotificationSink(consumer.HandleNotification)
	return consumer, cc, nil
}
