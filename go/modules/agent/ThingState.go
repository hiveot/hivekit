package agent

import (
	"maps"
	"sync"

	"github.com/hiveot/hivekit/go/api/msg"
)

// ThingState holds the state values of Things managed by the agent.
// Intended to support querying property and event values.
type ThingState struct {

	// Thing property values.
	properties map[string]any

	// Thing event values (not a WoT operation)
	// Updated when an event is published.
	events map[string]*msg.NotificationMessage

	// Last thing actions response
	// Updated when an action is processed
	actionResponse map[string]*msg.ResponseMessage

	mux sync.RWMutex
}

// Obtain the latest action response or nil if name is not a known action
func (tstate *ThingState) GetActionResponse(name string) *msg.ResponseMessage {
	tstate.mux.RLock()
	defer tstate.mux.RUnlock()
	resp, ok := tstate.actionResponse[name]
	_ = ok
	return resp
}

// Return a copy of all properties
func (tstate *ThingState) GetAllProperties() map[string]any {
	tstate.mux.RLock()
	defer tstate.mux.RUnlock()
	newMap := maps.Clone(tstate.properties)
	return newMap
}

// Return a copy of all events
func (tstate *ThingState) GetAllEvents() map[string]*msg.NotificationMessage {
	tstate.mux.RLock()
	defer tstate.mux.RUnlock()
	newMap := maps.Clone(tstate.events)
	return newMap
}

// Obtain the latest event notification or nil if name is not a known event
func (tstate *ThingState) GetEvent(name string) *msg.NotificationMessage {
	tstate.mux.RLock()
	defer tstate.mux.RUnlock()
	notif, ok := tstate.events[name]
	_ = ok
	return notif
}

// Obtain the latest property value
func (tstate *ThingState) GetProperty(name string) (value any, found bool) {
	tstate.mux.RLock()
	defer tstate.mux.RUnlock()
	val, ok := tstate.properties[name]
	return val, ok
}

// Set the latest action response by name
func (tstate *ThingState) SetActionResponse(name string, resp *msg.ResponseMessage) {
	tstate.mux.Lock()
	defer tstate.mux.Unlock()
	tstate.actionResponse[name] = resp
}

// Set the latest event of a name
func (tstate *ThingState) SetEvent(name string, notif *msg.NotificationMessage) {
	tstate.mux.Lock()
	defer tstate.mux.Unlock()
	tstate.events[name] = notif
}

// Set the latest property notification
func (tstate *ThingState) SetProperty(propName string, propVal any) {
	tstate.mux.Lock()
	defer tstate.mux.Unlock()
	tstate.properties[propName] = propVal
}

// Create a new instance of a thing state
func NewThingState(thingID string) *ThingState {
	tstate := &ThingState{
		properties:     make(map[string]any),
		events:         make(map[string]*msg.NotificationMessage),
		actionResponse: make(map[string]*msg.ResponseMessage),
	}
	return tstate
}
