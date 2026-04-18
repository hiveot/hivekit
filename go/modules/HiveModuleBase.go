package modules

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/teris-io/shortid"
)

// Module application environment
type ModuleEnv struct {
	// Application home directory
	HomeDirectory string
	// Application storage directory
	StorageDirectory string
}

// HiveModuleBase implements the boilerplate of running a module.
// This implements the IHiveModule interface.
// - define and store properties
// - manage message sinks
// - generate TD
// - send notifications for property changes and events
//
// Call Init(moduleID,sink) after construction
type HiveModuleBase struct {

	// notificationHandler is the application handler of notifications
	// notifications will also be forwarded upstream to the upstream handler.
	appNotificationHook msg.NotificationHandler

	// appRequestHook is the application handler of requests addressed to this module.
	//
	// HandleRequest will invoke this callback or forward requests not destined for
	// this module (moduleID != request.ThingID) to requestSink.
	appRequestHook msg.RequestHandler

	// Map of changed properties intended for sending property change notifications
	// This map is empty until changes are made using UpdateProperty
	// changedProperties map[string]any

	// notificationSink is the sink for forwarding notification messages
	// This is the upstream consumer.
	notificationSink msg.NotificationHandler

	// module properties and their value, nil if not used
	// use UpdateProperty to modify a value and flag it for change
	// properties map[string]any

	// RW mutex to access properties
	// propMux sync.RWMutex

	// requestSink is the sink for forwarding requests messages to
	requestSink msg.RequestHandler
}

// ForwardNotification (output) passes notifications to a registered hook and
// send it to the a registered sink.
//
// If none is registered this does nothing.
// note that the handler is not the downstream sink but the upstream consumer.
func (m *HiveModuleBase) ForwardNotification(notif *msg.NotificationMessage) {
	// why would this be here instead in HandleNotification?
	// if m.appNotificationHook != nil {
	// 	go m.appNotificationHook(notif)
	// }

	if m.notificationSink == nil {
		// End of the line. If the notification isn't handled then warn about it
		// A downstream module could have subscribed.
		if m.appNotificationHook == nil {
			// keep this warning for now.
			slog.Info("ForwardNotification: end of the line, no more notification sink.",
				"module", fmt.Sprintf("%T", m),
				"affordance", notif.AffordanceType,
				"thingID", notif.ThingID,
				"name", notif.Name,
			)
		}
		return
	}
	m.notificationSink(notif)
}

// ForwardRequest passes the request to the sink's HandleRequest method.
// If no sink os configured this returns an error
// This assigns a request correlationID if none is set.
func (m *HiveModuleBase) ForwardRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	if m.requestSink == nil {
		return fmt.Errorf("ForwardRequest: end of the line at '%s' for request '%s/%s' to thingID '%s'",
			fmt.Sprintf("%T", m), req.Operation, req.Name, req.ThingID)
	}
	if replyTo == nil {
		slog.Info("ForwardRequest: no replyTo handler provided")
	}
	err = m.requestSink(req, replyTo)
	return err
}

// ForwardRequestWait is a helper function to pass a request to the sink and wait for a response.
// If no sink os configured this returns an error.
// If the response contains an error, that error is also returned.
func (m *HiveModuleBase) ForwardRequestWait(req *msg.RequestMessage) (resp *msg.ResponseMessage, err error) {
	return msg.ForwardRequestWait(req, m.ForwardRequest)
}

// GetSink returns the module's request sink
func (m *HiveModuleBase) GetSink() msg.RequestHandler {
	return m.requestSink
}

// HandleNotification receives an incoming notification from a producer.
//
// The default behavior is to passes the notification to the registered hook and
// send it upstream to the registered notification handler, if set.
//
// Applications that consume notifications should use SetNotificationHook to register
// its handler as it leaves the chain intact..
func (m *HiveModuleBase) HandleNotification(notif *msg.NotificationMessage) {
	if m.appNotificationHook != nil {
		m.appNotificationHook(notif)
	}
	// the reason for the extra indirection is to ensure we're receiving the notification
	// independently from when someone sets a custome notification handler.
	// ForwardNotification will invoke the hook.
	m.ForwardNotification(notif)
}

// HandleRequest handles request for this module.
//
// This is just the default implementation. Applications can either set an appRequestHandler
// or a module can override HandleRequest to do its own thing.
//
// Modules that override HandleRequest should first handle the request itself and
// only hand it over to this base method when there is nothing for them to do. This method
// simply forwards the request if no request handler hook is set.
func (m *HiveModuleBase) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// Note, there is no thingID. So if the parent passes the request down and a request hook is set
	// then assume the handler will take care of forwarding the request as needed.
	if m.appRequestHook != nil {
		err = m.appRequestHook(req, replyTo)
		return err
	}

	return m.ForwardRequest(req, replyTo)
}

// Set the hook to invoke with received notifications
func (m *HiveModuleBase) SetNotificationHook(hook msg.NotificationHandler) {
	m.appNotificationHook = hook
}

// Set the handler that will receive notifications emitted by this module
func (m *HiveModuleBase) SetNotificationSink(consumer msg.NotificationHandler) {
	if m.notificationSink != nil {
		slog.Warn("SetNotificationSink: A notification sink already exists. It will be overwritten.")
	}
	m.notificationSink = consumer
}

// Set the hook to invoke with received requests directed at this module
// Any other requests received by HandleRequest will be forwarded to the sink.
func (m *HiveModuleBase) SetRequestHook(hook msg.RequestHandler) {
	m.appRequestHook = hook
}

// SetRequestSink sets the producer that will handle requests for this consumer and register this
// module as the receive of notifications from the module.
//
//	producer is the sink that will handle requests and send notifications
func (m *HiveModuleBase) SetRequestSink(sink msg.RequestHandler) {
	m.requestSink = sink
}

func (m *HiveModuleBase) Start() error {
	return nil
}
func (m *HiveModuleBase) Stop() {}

// UpdateProperty updates the given property value and sends a notification to subscribers.
// This tracks the changes to properties that can be retrieved with GetChangedProperties()
// func (m *HiveModuleBase) UpdateProperty(name string, val any) {
// 	m.propMux.Lock()
// 	if m.properties == nil {
// 		m.properties = make(map[string]any)
// 	}
// 	if m.changedProperties == nil {
// 		m.changedProperties = make(map[string]any)
// 	}
// 	m.properties[name] = val
// 	m.changedProperties[name] = val
// 	m.propMux.Unlock()

// 	//
// 	notif := msg.NewNotificationMessage(m.moduleID, msg.AffordanceTypeProperty, m.moduleID, name, val)
// 	m.ForwardNotification(notif)
// }
