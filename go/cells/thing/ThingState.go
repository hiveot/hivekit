package thing

import (
	"maps"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/utils"
)

// ExposedThingState holds the state values of an exposed Thing.
// Intended to support querying property and event values.
type ExposedThingState struct {

	// Thing property values
	properties map[string]msg.ThingValue

	// Thing event values
	// Updated when an event is published.
	events map[string]msg.ThingValue

	// Last action requested by client
	actions map[string]msg.ActionStatus

	mux sync.RWMutex
}

// Obtain the latest action status or nil if name is not a known action
func (tstate *ExposedThingState) GetActionStatus(name string) *msg.ActionStatus {
	tstate.mux.RLock()
	defer tstate.mux.RUnlock()
	req, ok := tstate.actions[name]
	if !ok {
		return nil
	}
	return &req
}

// Return a copy of all property values
func (tstate *ExposedThingState) GetAllProperties() map[string]msg.ThingValue {
	tstate.mux.RLock()
	defer tstate.mux.RUnlock()
	newMap := maps.Clone(tstate.properties)
	return newMap
}

// Return a copy of all events or nil if no events exist
func (tstate *ExposedThingState) GetAllEvents() map[string]msg.ThingValue {
	tstate.mux.RLock()
	defer tstate.mux.RUnlock()
	newMap := maps.Clone(tstate.events)
	return newMap
}

// Obtain the latest event notification or nil if name is not a known event
func (tstate *ExposedThingState) GetEvent(name string) *msg.ThingValue {
	tstate.mux.RLock()
	defer tstate.mux.RUnlock()
	evValue, ok := tstate.events[name]
	if !ok {
		return nil
	}
	return &evValue
}

// Obtain the latest property value
func (tstate *ExposedThingState) GetProperty(name string) *msg.ThingValue {
	tstate.mux.RLock()
	defer tstate.mux.RUnlock()
	val, ok := tstate.properties[name]
	if !ok {
		return nil
	}
	return &val
}

// Update the emitted event value
func (tstate *ExposedThingState) SetEvent(name string, value any) {
	tstate.mux.Lock()
	defer tstate.mux.Unlock()
	ts := utils.FormatUTCMilli(time.Now())
	tstate.events[name] = msg.ThingValue{
		Data:      value,
		Timestamp: ts,
	}
}

// Set the latest property value when received as a notification
func (tstate *ExposedThingState) SetProperty(propName string, propVal any) {
	tstate.mux.Lock()
	defer tstate.mux.Unlock()
	ts := utils.FormatUTCMilli(time.Now())
	val := msg.ThingValue{
		Data:      propVal,
		Timestamp: ts,
	}
	tstate.properties[propName] = val
}

// Create a new instance of a thing state
func NewThingState() *ExposedThingState {
	tstate := &ExposedThingState{
		properties: make(map[string]msg.ThingValue),
		events:     make(map[string]msg.ThingValue),
		actions:    make(map[string]msg.ActionStatus),
	}
	return tstate
}
