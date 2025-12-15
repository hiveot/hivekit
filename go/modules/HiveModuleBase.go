package modules

import (
	"github.com/hiveot/hivekit/go/modules/messaging"
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
type HiveModuleBase struct {
	// ModuleID/ThingID is the unique instance ID of this module.
	ModuleID string

	// module properties and their value
	Properties map[string]any

	// registers sinks for passing SME messages to
	sinks []IHiveModule
}

// AddSink sets the destination sink to forward messages to, to send the processing result to, or both.
// Modules can support a single or multiple sinks. If no more sinks can be added an error is returned.
// AddSink can be invoked before or after start is called.
func (m *HiveModuleBase) AddSink(sink IHiveModule) error {
	if m.sinks == nil {
		m.sinks = make([]IHiveModule, 0, 1)
	}
	m.sinks = append(m.sinks, sink)
	return nil
}

// GetTD returns the module's TD describing its properties, actions and events.
// If supported, the TD can be obtained after a successful start.
// If no TM is available then this returns "".
// Forms in the TD are typically added by the pipeline messaging server.
func (m *HiveModuleBase) GetTM() string {
	return ""
}

// HandleRequest handles the boilerplate request messages.
// This is the module input.
// Intended to be used as a sink for another module.
// If the request is not handled here then forward it to the sinks.
//
// Transport modules that receive requests from its clients should pass these to the
// sinks and NOT pass them to HandleRequests.
func (m *HiveModuleBase) HandleRequest(request *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
	switch request.Operation {
	// TODO: can this boilerplate be handled by a ModuleBase
	case wot.OpReadProperty:
		resp = m.ReadProperty(request)
	case wot.OpReadMultipleProperties:
		resp = m.ReadMultipleProperties(request)
	case wot.OpReadAllProperties:
		resp = m.ReadMultipleProperties(request)
	// directory specific operations could be handled here
	default:
		resp = m.SendRequest(request)
	}
	return resp
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
func (m *HiveModuleBase) HandleNotification(notif *messaging.NotificationMessage) {
	m.SendNotification(notif)
}

// ReadProperty returns a response containing the requested property value
func (m *HiveModuleBase) ReadProperty(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
	if m.Properties == nil {
		return nil
	}
	propValue, found := m.Properties[req.Name]
	if !found {
		return nil
	}
	resp = req.CreateResponse(propValue, nil)
	return resp
}

// ReadAllProperties returns a response containing the map of all known property values
func (m *HiveModuleBase) ReadAllProperties(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
	var propValueMap = make(map[string]any, 0)
	for k, v := range m.Properties {
		propValueMap[k] = v
	}
	resp = req.CreateResponse(propValueMap, nil)
	return resp
}

// ReadMultipleProperties returns a response containing the map of requested property values
func (m *HiveModuleBase) ReadMultipleProperties(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
	if m.Properties == nil || req.Input == nil {
		return nil
	}
	var propNames []string
	var propValueMap = make(map[string]any, 0)
	err := utils.DecodeAsObject(req.Input, &propNames)
	if err != nil {
		return req.CreateErrorResponse(err)
	}
	for _, propName := range propNames {
		propValue, found := m.Properties[propName]
		if found {
			propValueMap[propName] = propValue
		}
	}
	resp = req.CreateResponse(propValueMap, nil)
	return resp
}

// SendNotification is a helper function to pass notifications to all sinks
// This is the module output.
func (m *HiveModuleBase) SendNotification(notif *messaging.NotificationMessage) {
	if m.sinks == nil {
		return
	}
	for _, sink := range m.sinks {
		sink.HandleNotification(notif)
	}
}

// SendRequest is a helper function to pass requests to the sinks.
// If multiple sinks are registered then the first response is returned.
// This is the module output.
func (m *HiveModuleBase) SendRequest(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
	if m.sinks == nil {
		return nil
	}
	for _, sink := range m.sinks {
		resp = sink.HandleRequest(req)
		if resp != nil {
			return resp
		}
	}
	return nil
}

func (m *HiveModuleBase) Start() error {
	return nil
}
func (m *HiveModuleBase) Stop() {}

// UpdateProperty updates the given property value and sends a notification to the sinks.
func (m *HiveModuleBase) UpdateProperty(name string, val any) {
	if m.Properties == nil {
		m.Properties = make(map[string]any)
	}
	m.Properties[name] = val
	notif := messaging.NewNotificationMessage(wot.OpObserveProperty, m.ModuleID, name, val)
	m.SendNotification(notif)
}
