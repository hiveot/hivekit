package consumer

import (
	"errors"
	"sync"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	factory "github.com/hiveot/hivekit/go/modules/factory"
	"github.com/teris-io/shortid"
)

const ConsumerModuleType = "consumer"

// Consumer is a module representing a WoT consumer.
//
// This provides functions to read, write and observe properties, read and subscribe to
// events, and invoke and query actions.
//
// Usage:
//
//	This module can be used as a base for service clients that like to use the
//	ready-to-use API for sending requests and querying properties.
//
//	To use this consumer it needs to be linked to a transport client module in order to deliver requests
//	and receive notifications using one of the available transport protocols.
//
//	Linking can be done manually using SetRequestSink and SetNotificationSink, or
//	by including it as the first module in a recipe of the factory module.
type Consumer struct {
	// This consumer is a sink for the connection
	*modules.HiveModuleBase

	// The sink that will forward the requests and respond with notifications.
	// sink modules.IHiveModule

	// notificationHandler is the application handler of notifications
	// notifications will also be forwarded upstream to the upstream handler.
	appNotificationHook msg.NotificationHandler

	mux sync.RWMutex
}

// HandleNotification receives an notifications from a downstream module.
//
// If a appNotification handler is set then pass this to the handler and forward
// the notification to the upstream linked notification handler.
//
// See also SetNotificationHook to allow applications to register a handler that receives
// notifications passing through the chain.
func (m *Consumer) HandleNotification(notif *msg.NotificationMessage) {
	m.mux.RLock()
	handler := m.appNotificationHook
	m.mux.RUnlock()

	if handler != nil {
		handler(notif)
	}
	// the reason for the extra indirection is to ensure we're receiving the notification
	// independently from when someone sets a custome notification handler.
	// ForwardNotification will invoke the hook.
	m.ForwardNotification(notif)
}

// InvokeAction invokes an action on a thing and wait for the response
// If the response type is known then provide it with output, otherwise use interface{}
func (co *Consumer) InvokeAction(
	thingID, name string, input any, output any) error {

	err := co.Rpc(td.OpInvokeAction, thingID, name, input, output)
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

	err := co.Rpc(op, thingID, name, nil, "")
	return err
}

// Ping the server and wait for a response.
// Intended to ensure the server is reachable.
func (co *Consumer) Ping() (err error) {
	var value any

	err = co.Rpc(td.HTOpPing, "", "", nil, &value)
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

	err = co.Rpc(td.OpQueryAction, thingID, name, nil, &value)
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

	err = co.Rpc(td.OpQueryAllActions, thingID, "", nil, &values)
	return values, err
}

// ReadAllEvents sends a request to read all Thing event values from the hub.
//
// This returns a map of eventName and the last sent notification message.
func (co *Consumer) ReadAllEvents(thingID string) (
	values map[string]*msg.NotificationMessage, err error) {

	err = co.Rpc(td.HTOpReadAllEvents, thingID, "", nil, &values)
	return values, err
}

// ReadAllProperties sends a request to read all Thing property values.
//
// This returns a map of property name-value pairs as described in the TD.
func (co *Consumer) ReadAllProperties(thingID string) (
	values map[string]any, err error) {

	err = co.Rpc(td.OpReadAllProperties, thingID, "", nil, &values)
	return values, err
}

// ReadEvent sends a request to read the last event message sent by a Thing.
//
// This returns the NotificationMessage that was last sent, containing the timestamp
// and value as described in the event affordance.
func (co *Consumer) ReadEvent(thingID, name string) (value *msg.NotificationMessage, err error) {

	err = co.Rpc(td.HTOpReadEvent, thingID, name, nil, &value)
	return value, err
}

// ReadProperty sends a request to read the current value of a Thing property.
//
// This decodes the value into the provided type
func (co *Consumer) ReadProperty(thingID, name string, output any) (err error) {

	err = co.Rpc(td.OpReadProperty, thingID, name, nil, output)
	return err
}

// ReadPropertyAs sends a request to read the current value of a Thing property.
//
// This converts the property value to the given type or returns an error
func (co *Consumer) ReadPropertyAs(thingID, name string, prop any) (err error) {

	err = co.Rpc(td.OpReadProperty, thingID, name, nil, prop)
	return err
}

// Set the hook to invoke with received notifications
//
// This lets applications receive notifications while leaving the notification chain intact.
func (m *Consumer) SetNotificationHook(hook msg.NotificationHandler) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.appNotificationHook = hook
}

// Subscribe to one or all events of a thing.
// name is the event to subscribe to or "" for all events
func (co *Consumer) Subscribe(thingID string, name string) error {
	op := td.OpSubscribeEvent
	if name == "" {
		op = td.OpSubscribeAllEvents
	}
	err := co.Rpc(op, thingID, name, nil, "")
	return err
}

// UnobserveProperty a previous observed property or all properties
func (co *Consumer) UnobserveProperty(thingID string, name string) error {
	op := td.OpUnobserveProperty
	if name == "" {
		op = td.OpUnobserveAllProperties
	}
	err := co.Rpc(op, thingID, name, nil, "")
	return err
}

// Unsubscribe is a helper for sending an unsubscribe request
func (co *Consumer) Unsubscribe(thingID string, name string) error {
	op := td.OpUnsubscribeEvent
	if name == "" {
		op = td.OpUnsubscribeAllEvents
	}
	err := co.Rpc(op, thingID, name, nil, "")
	return err
}

// WriteProperty is a helper to send a write property request
// Since writing properties can take some time on slow devices, the wait is optional.
func (co *Consumer) WriteProperty(thingID string, name string, input any, wait bool) (err error) {
	correlationID := shortid.MustGenerate()
	if wait {
		err = co.Rpc(td.OpWriteProperty, thingID, name, input, correlationID)
	} else {
		req := msg.NewRequestMessage(td.OpWriteProperty, thingID, name, input)
		req.CorrelationID = correlationID
		err = co.ForwardRequest(req, func(resp *msg.ResponseMessage) error {
			// just ignore the result
			return nil
		})
	}
	return err
}

// NewConsumer returns a new instance of the WoT consumer.
//
// This provides the API for common WoT operations such as invoking actions and
// supports RPC calls by waiting for a response.
//
// A notification handler can be provided or set with SetNotificationHook
// Use SetTimeout to modify the default RPC timeout
func NewConsumer(notificationHook msg.NotificationHandler) *Consumer {
	thingID := ConsumerModuleType + "-" + shortid.MustGenerate()
	consumer := &Consumer{
		HiveModuleBase:      modules.NewHiveModuleBase(thingID, msg.DefaultRnRTimeout),
		appNotificationHook: notificationHook,
	}

	return consumer
}

// Factory for creating a consumer module using the factory environment
func NewConsumerFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	c := NewConsumer(nil)
	c.SetTimeout(f.GetEnvironment().RpcTimeout)
	return c, nil
}
