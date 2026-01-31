package clients

import (
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
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
// Consumers can register callbacks for receiving notifications and changes in the connection.
//
// This implements the IHiveModule interface so it can be used as a sink for transports
// or other modules.
type Consumer struct {
	// This consumer is a sink for the connection
	modules.HiveModuleBase

	appID string

	// The sink that will forward the requests and respond with notifications.
	// sink modules.IHiveModule

	mux sync.RWMutex

	// The timeout to use when waiting for a response
	rpcTimeout time.Duration
}

// GetClientID returns the client's account ID
func (co *Consumer) GetClientID() string {
	return co.GetModuleID()
}

// GetModuleID returns the application
func (co *Consumer) GetModuleID() string {
	return co.appID
}

// GetTM returns empty
func (co *Consumer) GetTM() string {
	return ""
}

func (co *Consumer) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	return fmt.Errorf("Unexpected request op='%s', thingID='%s', name='%s', from '%s'",
		req.Operation, req.ThingID, req.Name, req.SenderID)
}

// InvokeAction invokes an action on a thing and wait for the response
// If the response type is known then provide it with output, otherwise use interface{}
func (co *Consumer) InvokeAction(
	thingID, name string, input any, output any) error {

	err := co.Rpc(wot.OpInvokeAction, thingID, name, input, output)
	return err
}

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

// handle incoming notifications from the sink
//
// If this consumer has a notification handler set (eg when used as a sink itself)
// then pass the notification to this handler.
// This logs an error if the consumer does not have a notification handler set, as
// notifications are only received when subscribed something must have gone wrong.
func (co *Consumer) onNotification(notif *msg.NotificationMessage) {
	co.ForwardNotification(notif)
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
// If replyTo is nil then responses are ignored.
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
	err = co.ForwardRequest(req, func(resp *msg.ResponseMessage) error {
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
			slog.Info("SendRequest: no response handler provided")
		}
		return err2
	})
	return err
}

// Start using the consumer
// this module does not have a configuration
func (co *Consumer) Start(yamlConfig string) error {
	return nil
}

// Stop the consumer module and closes the client connection.
func (co *Consumer) Stop() {
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

// NewConsumer returns a new instance of the WoT consumer for use with the given
// connection.
//
// This provides the API for common WoT operations such as invoking actions and
// supports RPC calls by waiting for a response.
//
// Use SetSink to set the module that will handle requests and return notifications.
//
//	appID the ID of this application
//	timeout of the rpc connections or 0 for default (3 sec)
func NewConsumer(appID string, rpcTimeout time.Duration) *Consumer {
	if rpcTimeout == 0 {
		rpcTimeout = transports.DefaultRpcTimeout
	}
	consumer := &Consumer{
		// sink:  sink,
		appID: appID,
		// rnrChan:    NewRnRChan(),
		rpcTimeout: rpcTimeout,
	}

	consumer.SetModuleID(appID)
	return consumer
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

	cc, err := NewClientModule(serverURL, caCert, rpcTimeout)
	if err != nil {
		return nil, nil, err
	}
	// set the connection as the sink that handles requests and publishes notifications
	consumer := NewConsumer(appID, rpcTimeout)
	consumer.SetRequestSink(cc.HandleRequest)
	cc.SetNotificationSink(consumer.HandleNotification)
	return consumer, cc, nil
}
