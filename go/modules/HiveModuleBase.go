package modules

import (
	"github.com/hiveot/hivekit/go/lib/messaging"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
)

// HiveModuleBase implements the boilerplate of running a module.
// - define and store properties
// - manage message sinks
// - generate TD
// - send notifications for property changes and events
type HiveModuleBase struct {
	// ThingID of this module instance
	ThingID string

	// properties and their value
	Properties map[string]any

	// registers sinks for receiving update messages
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
// If no TD is supported then this returns nil.
// Forms in the TD are typically added by the pipeline messaging server.
func (m *HiveModuleBase) GetTD() *td.TD {
	// construct a default TD using ID and properties
	td := td.NewTD(m.ThingID, "HiveKit Module "+m.ThingID, "module")

	return td
}

// HandleRequest handles the boilerplate request messages.
// If the request is not boilerplate then forward it to the sinks.
// Intended to be called by modules if the request is not for their implementation.
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

// HandleNotification forwards a notification message to the sinks.
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

// SendNotification passes a notification to all sinks
func (m *HiveModuleBase) SendNotification(notif *messaging.NotificationMessage) {
	if m.sinks == nil {
		return
	}
	for _, sink := range m.sinks {
		sink.HandleNotification(notif)
	}
}

// UpdateProperty updates the given property value and sends a notification to the sinks.
func (m *HiveModuleBase) UpdateProperty(name string, val any) {
	if m.Properties == nil {
		m.Properties = make(map[string]any)
	}
	m.Properties[name] = val
	notif := messaging.NewNotificationMessage(wot.OpObserveProperty, m.ThingID, name, val)
	m.SendNotification(notif)
}

// SendRequest passes a request message to the sinks.
// If multiple sinks are registered then the first response is returned.
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
