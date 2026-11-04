package internal

import (
	"sync"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/utils"
)

// ThingValues stores the latest event and property values of a thing
type ThingValues struct {
	Actions    map[string]msg.ActionStatus `json:"actions"`
	Events     map[string]msg.ThingValue   `json:"events"`
	Properties map[string]msg.ThingValue   `json:"properties"`
}

// ValueStore is an in-memory value-cache
type ValueStore struct {
	mux           sync.RWMutex
	ThingValueMap map[string]ThingValues `json:"things"`
}

// Return the nr of things with cached values
func (store *ValueStore) GetNrThings() int {
	store.mux.RLock()
	defer store.mux.RUnlock()
	return len(store.ThingValueMap)
}

// Return the latest cached action status or nil if not found
func (store *ValueStore) ReadAction(thingID string, name string) (actionStatus msg.ActionStatus, found bool) {
	store.mux.RLock()
	defer store.mux.RUnlock()
	thingValues, found := store.ThingValueMap[thingID]
	if found {
		actionStatus, found = thingValues.Actions[name]
	}
	return actionStatus, found
}

// Return the latest cached event or nil if not found
func (store *ValueStore) ReadEvent(thingID string, name string) (tv msg.ThingValue, found bool) {
	store.mux.RLock()
	defer store.mux.RUnlock()
	thingValues, found := store.ThingValueMap[thingID]
	if found {
		tv, found = thingValues.Events[name]
	}
	return tv, found
}

// Return the cached value of a property
func (store *ValueStore) ReadProperty(thingID string, name string) (tv msg.ThingValue, found bool) {
	store.mux.RLock()
	defer store.mux.RUnlock()
	thingValues, found := store.ThingValueMap[thingID]
	if found {
		tv, found = thingValues.Properties[name]
	}
	return tv, found
}

// Return the cached value of multiple properties
// allFound is false if not all of them are found
func (store *ValueStore) ReadMultipleProperties(
	thingID string, names []string) (tvMap map[string]msg.ThingValue, allFound bool) {

	store.mux.RLock()
	defer store.mux.RUnlock()
	tv, allFound := store.ThingValueMap[thingID]
	if allFound {
		tvMap = make(map[string]msg.ThingValue)
		for _, name := range names {
			prop, hasProp := tv.Properties[name]
			if hasProp {
				tvMap[name] = prop
			} else {
				allFound = false
			}
		}
	}
	return tvMap, allFound
}

// Remove a property value from the cache
func (store *ValueStore) RemoveProperty(thingID string, name string) {
	store.mux.Lock()
	defer store.mux.Unlock()
	tv, found := store.ThingValueMap[thingID]
	if found {
		delete(tv.Properties, name)
	}
}

// Write the value from the notification
func (store *ValueStore) WriteValue(notif *msg.NotificationMessage) {
	store.mux.Lock()
	defer store.mux.Unlock()
	thingValues, found := store.ThingValueMap[notif.ThingID]
	if !found {
		thingValues = ThingValues{
			Actions:    make(map[string]msg.ActionStatus),
			Events:     make(map[string]msg.ThingValue),
			Properties: make(map[string]msg.ThingValue),
		}
	}
	switch notif.AffordanceType {
	case msg.AffordanceTypeAction:
		var actionStatus msg.ActionStatus
		utils.Decode(notif.Data, &actionStatus)
		thingValues.Actions[notif.Name] = actionStatus
	case msg.AffordanceTypeEvent:
		tv := msg.ThingValue{
			Data:      notif.Data,
			Timestamp: notif.Timestamp,
		}
		thingValues.Events[notif.Name] = tv
	case msg.AffordanceTypeProperty:
		tv := msg.ThingValue{
			Data:      notif.Data,
			Timestamp: notif.Timestamp,
		}
		thingValues.Properties[notif.Name] = tv
	}
	store.ThingValueMap[notif.ThingID] = thingValues
}

func NewVCacheStore() *ValueStore {
	store := &ValueStore{
		ThingValueMap: make(map[string]ThingValues),
	}
	return store
}
