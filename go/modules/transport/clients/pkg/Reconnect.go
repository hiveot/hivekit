package clientspkg

import (
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
)

const ReconnectModuleType = "reconnect"

// Reconnect is a module that automatically re-applies event subscriptions and property
// observations after a connection is restored.
//
// # Place this module before the connection client module in the chain
//
// For this to work the following rules must be enforced to client modules:
//   - client modules must submit disconnect and connect notifications
//   - consumers must subscribe using the standard TD operations for subscriptions.
//
// Future consideration:
//   - have client modules connect/disconnect using RRN messaging instead of direct API
//     and implement the reconnect attempt in this module.
type Reconnect struct {
	modules.HiveModuleBase

	// record of subscriptions by key="{thingID}-{name}"
	subscriptions map[string]*msg.RequestMessage
}

// HandleNotification detects a disconnect and reconnect from a client module.

func (m *Reconnect) HandleNotification(notif *msg.NotificationMessage) {
	// TODO:
	m.HiveModuleBase.HandleNotification(notif)
}

// HandleRequest tracks subscriptions to events and property updates
func (m *Reconnect) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	switch req.Operation {
	case td.OpSubscribeAllEvents, td.OpSubscribeEvent,
		td.OpObserveAllProperties, td.OpObserveMultipleProperties, td.OpObserveProperty:

		// TBD: this doesn't differentiate between event/property affordance or single or multiple
		// TODO: how to handle subscription to multiple properties?
		key := fmt.Sprintf("%s-%s", req.ThingID, req.Name)
		m.subscriptions[key] = req

	case td.OpUnobserveAllProperties, td.OpUnobserveMultipleProperties, td.OpUnobserveProperty,
		td.OpUnsubscribeAllEvents, td.OpUnsubscribeEvent:
		// remove the recorded subscription request
		// TODO: remove all on a disconnect request
		key := fmt.Sprintf("%s-%s", req.ThingID, req.Name)
		delete(m.subscriptions, key)
	}
	// forward
	return m.HiveModuleBase.HandleRequest(req, replyTo)
}

// NewReconnect returns a new instance of the client auto-reconnect module.
//
//	timeout is the maximum time to wait to reconnect or 0 for the default
func NewReconnect(timeout time.Duration) *Reconnect {
	m := &Reconnect{
		HiveModuleBase: modules.NewHiveModuleBase("Reconnect", timeout),
		subscriptions:  make(map[string]*msg.RequestMessage),
	}

	return m
}

// Factory for creating a consumer module using the factory environment
func NewReconnectFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	c := NewReconnect(f.GetEnvironment().RpcTimeout)
	return c, nil
}
