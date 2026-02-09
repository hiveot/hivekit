package modules

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
)

// Module application environment
type ModuleEnv struct {
	// Application home directory
	HomeDirectory string
	// Application storage directory
	StorageDirectory string
}

// HiveModuleBase implements the boilerplate of running a module.
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

	// requestHandler is the application handler of requests addressed to this module.
	//
	// HandleRequest will invoke this callback or forward requests not destined for
	// this module (moduleID != request.ThingID) to requestSink.
	appRequestHandler msg.RequestHandler

	// Map of changed properties intended for sending property change notifications
	// This map is empty until changes are made using UpdateProperty
	changedProperties map[string]any

	// moduleID/thingID is the unique instance ID of this module.
	moduleID string

	// notificationSink is the sink for forwarding notification messages
	// This is the upstream consumer.
	notificationSink msg.NotificationHandler

	// module properties and their value, nil if not used
	// use UpdateProperty to modify a value and flag it for change
	properties map[string]any

	// RW mutex to access properties
	propMux sync.RWMutex

	// requestSink is the sink for forwarding requests messages to
	requestSink msg.RequestHandler
}

// ForwardNotification (output) passes notifications to the registered callback.
// If none is registered this does nothing.
// note that the handler is not the downstream sink but the upstream consumer.
func (m *HiveModuleBase) ForwardNotification(notif *msg.NotificationMessage) {
	if m.notificationSink == nil {
		// End of the line. A downstream module could have subscribed.
		// Should subscribers not have a direct callback? probably... tbd

		// keep this warning for now.
		slog.Warn("ForwardNotification: no handler set. Notification is dropped.",
			"module", m.moduleID,
			"affordance", notif.AffordanceType,
			"thingID", notif.ThingID,
			"name", notif.Name,
		)
		return
	}
	m.notificationSink(notif)
}

// ForwardRequest (output) is a helper function to pass a request to the sink's
// HandleRequest method.
// If no sink os configured this returns an error
func (m *HiveModuleBase) ForwardRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if m.requestSink == nil {
		return fmt.Errorf("ForwardRequest: no sink for request '%s/%s' to thingID '%s'",
			req.Operation, req.Name, req.ThingID)
	}
	err = m.requestSink(req, replyTo)
	return err
}

// ForwardRequestWait is a helper function to pass a request to the sink and wait for a response.
// If no sink os configured this returns an error.
// If the response contains an error, that error is also returned.
func (m *HiveModuleBase) ForwardRequestWait(req *msg.RequestMessage) (resp *msg.ResponseMessage, err error) {
	ar := utils.NewAsyncReceiver[*msg.ResponseMessage]()

	err = m.ForwardRequest(req, func(r *msg.ResponseMessage) error {
		ar.SetResponse(r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	resp, err = ar.WaitForResponse(0)
	if err == nil {
		err = resp.AsError()
	}
	return resp, err
}

// GetModuleID returns the module's Thing ID
func (m *HiveModuleBase) GetModuleID() string {
	return m.moduleID
}

// GetSink returns the module's request sink
func (m *HiveModuleBase) GetSink() msg.RequestHandler {
	return m.requestSink
}

// GetTM returns the module's TM describing its properties, actions and events.
// If supported, the TM can be obtained after a successful start.
// If no TM is available then this returns "".
// To convert the TM to a TD, use AddForms on the transport modules to include
// forms that describe interactions. This can be handled by the pipeline server
// or by the application itself.
func (m *HiveModuleBase) GetTM() string {
	return ""
}

// HandleNotification receives an incoming notification from a producer.
//
// The default behavior passes the notification to the registered hook an
// forwards it upstream to a register notification handler, if set.
//
// Applications that use notifications should use SetNotificationHook to register
// its handler as it leaves the chain intact..
func (m *HiveModuleBase) HandleNotification(notif *msg.NotificationMessage) {
	if m.appNotificationHook != nil {
		go m.appNotificationHook(notif)
	}
	// the reason for the extra indirection is to ensure we're receiving the notification
	// independently from when someone sets a custome notification handler.
	m.ForwardNotification(notif)
}

// HandleRequest handles request for this module.
//
// This is just the default implementation. Applications can either set an appRequestHandler
// or a module can override HandleRequest to do its own thing.
//
// If appRequestHandler is set, the request is passed to the application.
//
// Modules that override HandleRequest should first handle the request itself and
// only hand it over to this base method when there is nothing for them to do.
//
// If a custom request handler is set then this invokes that handler. Intended for
// use by applications (and tests) that implement a function as request handler.
//
// ReadProperties is handled here as a convenience.
//
// If the request is not for this module, it is forwarded to the sink. if defined.
// If the request is for this module and read property(ies), it is handled here.
//
// If the request is unhandled it returns an error.
func (m *HiveModuleBase) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var resp *msg.ResponseMessage

	if m.appRequestHandler != nil {
		err = m.appRequestHandler(req, replyTo)
		return err
	}

	if req.ThingID != m.moduleID {
		return m.ForwardRequest(req, replyTo)
	}
	// handle the read property requests
	switch req.Operation {

	case wot.OpReadProperty:
		resp, err = m.ReadProperty(req)
	case wot.OpReadMultipleProperties:
		resp, err = m.ReadMultipleProperties(req)
	case wot.OpReadAllProperties:
		resp, err = m.ReadAllProperties(req)
		// directory specific operations could be handled here
	default:
		err := fmt.Errorf("Unhandled request: thingID='%s', op='%s', name='%s", req.ThingID, req.Operation, req.Name)
		slog.Warn(err.Error())
	}
	if resp != nil {
		err = replyTo(resp)
	}
	return err
}

// ReadAllProperties returns a response containing the map of all known property values
func (m *HiveModuleBase) ReadAllProperties(req *msg.RequestMessage) (resp *msg.ResponseMessage, err error) {
	m.propMux.RLock()
	var propValueMap = make(map[string]any, 0)
	for k, v := range m.properties {
		propValueMap[k] = v
	}
	m.propMux.RUnlock()
	resp = req.CreateResponse(propValueMap, nil)
	return resp, err
}

// ReadChangedProperties returns the changed properties and clear the tracked changes
// Intended to be used with sending a notification of changed properties.
// This returns nil if no properties have changed.
func (m *HiveModuleBase) ReadChangedProperties() (changes map[string]any) {
	m.propMux.Lock()
	defer m.propMux.Unlock()

	changes = m.changedProperties
	m.changedProperties = nil
	return changes
}

// ReadMultipleProperties returns a response containing the map of requested property values
// If a requested property doesn't exist then it isn't included in the result. This
// should be considered an error but not reason enough to fail reading the other properties.
func (m *HiveModuleBase) ReadMultipleProperties(req *msg.RequestMessage) (resp *msg.ResponseMessage, err error) {
	var propValueMap = make(map[string]any, 0)

	if m.properties != nil || req.Input != nil {
		var propNames []string
		err = utils.DecodeAsObject(req.Input, &propNames)
		if err != nil {
			resp = req.CreateErrorResponse(err)
		} else {
			m.propMux.RLock()
			for _, propName := range propNames {
				propValue, found := m.properties[propName]
				if found {
					propValueMap[propName] = propValue
				} else {
					// while this is an error, there is no reason to fail the whole request.
				}
			}
			m.propMux.RUnlock()
			resp = req.CreateResponse(propValueMap, nil)
		}
	}
	return resp, err
}

// ReadProperty returns a response containing the requested property value
// This returns an error if the property doesn't exist.
func (m *HiveModuleBase) ReadProperty(req *msg.RequestMessage) (resp *msg.ResponseMessage, err error) {
	var found bool
	var propValue any

	m.propMux.RLock()
	if m.properties == nil {
		found = false
	} else {
		propValue, found = m.properties[req.Name]
	}
	m.propMux.RUnlock()
	if !found {
		err = fmt.Errorf("Property '%s' doesn't exist on Thing '%s'", req.Name, req.ThingID)
		return nil, err
	}
	resp = req.CreateResponse(propValue, nil)
	return resp, err
}

// Initialize the module base with a moduleID
//
// This sets this module notification handler with the sink.
func (m *HiveModuleBase) SetModuleID(moduleID string) {
	m.moduleID = moduleID
}

// Set the hook to invoke with received notifications
func (m *HiveModuleBase) SetNotificationHook(hook msg.NotificationHandler) {
	m.appNotificationHook = hook
}

// Set the handler that will receive notifications emitted by this module
func (m *HiveModuleBase) SetNotificationSink(consumer msg.NotificationHandler) {
	m.notificationSink = consumer
}

// Set the hook to invoke with received requests directed at this module
// Any other requests received by HandleRequest will be forwarded to the sink.
func (m *HiveModuleBase) SetRequestHook(hook msg.RequestHandler) {
	m.appRequestHandler = hook
}

// SetRequestSink sets the producer that will handle requests for this consumer and register this
// module as the receive of notifications from the module.
//
//	producer is the sink that will handle requests and send notifications
func (m *HiveModuleBase) SetRequestSink(sink msg.RequestHandler) {
	m.requestSink = sink
}

func (m *HiveModuleBase) Start(yamlConfig string) error {
	return nil
}
func (m *HiveModuleBase) Stop() {}

// UpdateProperty updates the given property value and sends a notification to subscribers.
// This tracks the changes to properties that can be retrieved with GetChangedProperties()
func (m *HiveModuleBase) UpdateProperty(name string, val any) {
	m.propMux.Lock()
	if m.properties == nil {
		m.properties = make(map[string]any)
	}
	if m.changedProperties == nil {
		m.changedProperties = make(map[string]any)
	}
	m.properties[name] = val
	m.changedProperties[name] = val
	m.propMux.Unlock()

	//
	notif := msg.NewNotificationMessage(m.moduleID, msg.AffordanceTypeProperty, m.moduleID, name, val)
	m.ForwardNotification(notif)
}
