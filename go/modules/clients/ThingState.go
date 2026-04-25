package clients

import (
	"sync"

	"github.com/hiveot/hivekit/go/api/msg"
)

// ThingState holds the state values of a thing managed by the agent.
// Intended to support querying property and event values.
type ThingState struct {

	// Thing property values.
	// Updated when a property is published.
	properties map[string]*msg.NotificationMessage

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

// Obtain the latest event notification or nil if name is not a known event
func (tstate *ThingState) GetEvent(name string) *msg.NotificationMessage {
	tstate.mux.RLock()
	defer tstate.mux.RUnlock()
	notif, ok := tstate.events[name]
	_ = ok
	return notif
}

// Obtain the latest property notification or nil if name is not a known property
func (tstate *ThingState) GetProperty(name string) *msg.NotificationMessage {
	tstate.mux.RLock()
	defer tstate.mux.RUnlock()
	notif, ok := tstate.properties[name]
	_ = ok
	return notif
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

// Set the latest event of a name
func (tstate *ThingState) SetProperty(name string, notif *msg.NotificationMessage) {
	tstate.mux.Lock()
	defer tstate.mux.Unlock()
	tstate.properties[name] = notif
}

// Create a new instance of a thing state
func NewThingState(thingID string) *ThingState {
	tstate := &ThingState{
		properties:     make(map[string]*msg.NotificationMessage),
		events:         make(map[string]*msg.NotificationMessage),
		actionResponse: make(map[string]*msg.ResponseMessage),
	}
	return tstate
}
