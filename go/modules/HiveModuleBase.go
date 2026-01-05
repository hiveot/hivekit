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
	// moduleID/thingID is the unique instance ID of this module.
	moduleID string

	// module properties and their value, nil if not used
	// use UpdateProperty to modify a value and flag it for change
	properties map[string]any

	// Map of changed properties intended for sending property change notifications
	// This map is empty until changes are made using UpdateProperty
	changedProperties map[string]any

	// RW mutex to access properties
	propMux sync.RWMutex

	// Output/sink for forwarding RRN messages to
	sink IHiveModule
}

// SetSink sets the destination sink to forward messages to.
// This overwrites an existing sink if already set
// func (m *HiveModuleBase) SetSink(sink IHiveModule) {
// 	m.sink = sink
// }

// ForwardNotification (output) is a helper function to pass notifications to all sinks
func (m *HiveModuleBase) ForwardNotification(notif *msg.NotificationMessage) {
	if m.sink == nil {
		return
	}
	m.sink.HandleNotification(notif)
}

// ForwardRequest (output) is a helper function to pass a request to the sink's
// HandleRequest method.
// If no sink os configured this returns an error
func (m *HiveModuleBase) ForwardRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if m.sink == nil {
		return fmt.Errorf("no sink for request '%s/%s' to thingID '%s'",
			req.Operation, req.Name, req.ThingID)
	}
	err = m.sink.HandleRequest(req, replyTo)
	return err
}

// ForwardResponse (output) is a helper function to pass a response to the sink
// for further forwarding. The sinks will receive this as HandleResponse.
//
// The primary use-case for this is when a transport module receives a response,
// to send it down the chain until a module catches it.
//
// Alternatively, responses from transport modules can be passed to a router, in
// which case this isn't used. This depends on the pipeline configuration.
//
// This returns the result of forwarding or an error if no sinks are registered
func (m *HiveModuleBase) ForwardResponse(resp *msg.ResponseMessage) (err error) {
	if m.sink == nil {
		return fmt.Errorf("end of the line. No more sink to forward response by agent '%s', from ThingID '%s', op '%s'.",
			resp.SenderID, resp.ThingID, resp.Operation)
	}
	err = m.sink.HandleResponse(resp)
	return err
}

// GetModuleID returns the module's Thing ID
func (m *HiveModuleBase) GetModuleID() string {
	return m.moduleID
}

// GetSink returns the module's output handler
func (m *HiveModuleBase) GetSink() IHiveModule {
	return m.sink
}

// GetTD returns the module's TD describing its properties, actions and events.
// If supported, the TD can be obtained after a successful start.
// If no TM is available then this returns "".
// Forms in the TD are typically added by the pipeline messaging server.
func (m *HiveModuleBase) GetTM() string {
	return ""
}

// HandleNotification process an incoming notification.
// This is the module input.
//
// The default behavior is to forward notifications to the sinks, so it is part
// of the pipeline chain.
//
// In transport modules, notifications should be passed to connected clients that have
// subscribed to the notification.
//
// Transport modules that receive notifications from its clients should pass these to the
// sinks and NOT pass them to HandleNotification.
func (m *HiveModuleBase) HandleNotification(notif *msg.NotificationMessage) {
	m.ForwardNotification(notif)
}

// HandleRequest handles the read property request for this module.
//
// Intended for use by the module's HandleRequest itself. This handles the reading properties boilerplate.
// Modules should first handle the request itself and only hand it over to this base method when
// there is nothing for them to do.
//
// If the request is for another module, the request is forwarded to the sink, if defined.
//
// If the request is unhandled it returns an error.
func (m *HiveModuleBase) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var resp *msg.ResponseMessage
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
		resp, err = m.ReadMultipleProperties(req)
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

// HandleResponse receives a response for processing or forwarding.
// Handling responses are consumer activities.
//
// When addressed to this module, it is ignored as there is nothing to do.
// Subclasses should handle the response if an output is expected.
//
// When not addressed to this module, the response might be intended for a module
// down the chain, so forward it.
func (m *HiveModuleBase) HandleResponse(resp *msg.ResponseMessage) error {
	if resp.ThingID == m.moduleID {
		return nil
	}
	return m.ForwardResponse(resp)
}

// Initialize the module base with a moduleID and a messaging sink
func (m *HiveModuleBase) Init(moduleID string, sink IHiveModule) {
	m.moduleID = moduleID
	m.sink = sink
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

func (m *HiveModuleBase) Start() error {
	return nil
}
func (m *HiveModuleBase) Stop() {}

// UpdateProperty updates the given property value and sends a notification to the sinks.
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
	notif := msg.NewNotificationMessage(wot.OpObserveProperty, m.moduleID, name, val)
	m.ForwardNotification(notif)
}
