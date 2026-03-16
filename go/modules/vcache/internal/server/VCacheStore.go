package module

import (
	"sync"

	"github.com/hiveot/hivekit/go/msg"
)

// ThingNotifications stores the latest event and property values of a thing
type ThingNotifications struct {
	Actions    map[string]*msg.NotificationMessage `json:"actions"`
	Events     map[string]*msg.NotificationMessage `json:"events"`
	Properties map[string]*msg.NotificationMessage `json:"properties"`
}

type VCacheStore struct {
	mux    sync.RWMutex
	Things map[string]ThingNotifications `json:"things"`
}

// Return the nr of things with cached values
func (store *VCacheStore) GetNrThings() int {
	store.mux.RLock()
	defer store.mux.RUnlock()
	return len(store.Things)
}

// Return the latest cached action status or nil if not found
func (store *VCacheStore) ReadAction(thingID string, name string) (action *msg.NotificationMessage) {
	store.mux.RLock()
	defer store.mux.RUnlock()
	tn, found := store.Things[thingID]
	if found {
		action, found = tn.Actions[name]
	}
	return action
}

// Return the latest cached event or nil if not found
func (store *VCacheStore) ReadEvent(thingID string, name string) (event *msg.NotificationMessage) {
	store.mux.RLock()
	defer store.mux.RUnlock()
	tn, found := store.Things[thingID]
	if found {
		event, found = tn.Events[name]
	}
	return event
}

// Return the cached value of a property or nil if not found
func (store *VCacheStore) ReadProperty(thingID string, name string) (prop *msg.NotificationMessage) {
	store.mux.RLock()
	defer store.mux.RUnlock()
	tv, found := store.Things[thingID]
	if found {
		prop, found = tv.Properties[name]
	}
	return prop
}

// Return the cached value of multiple properties
// allFound is false if not all of them are found
func (store *VCacheStore) ReadMultipleProperties(
	thingID string, names []string) (notifMap map[string]*msg.NotificationMessage, allFound bool) {

	store.mux.RLock()
	defer store.mux.RUnlock()
	tv, allFound := store.Things[thingID]
	if allFound {
		notifMap = make(map[string]*msg.NotificationMessage)
		for _, name := range names {
			prop, hasProp := tv.Properties[name]
			if hasProp {
				notifMap[name] = prop
			} else {
				allFound = false
			}
		}
	}
	return notifMap, allFound
}

// Remove a property value from the cache
func (store *VCacheStore) RemoveProperty(thingID string, name string) {
	store.mux.Lock()
	defer store.mux.Unlock()
	tv, found := store.Things[thingID]
	if found {
		delete(tv.Properties, name)
	}
}

func (store *VCacheStore) WriteValue(notif *msg.NotificationMessage) {
	store.mux.Lock()
	defer store.mux.Unlock()
	tv, found := store.Things[notif.ThingID]
	if !found {
		tv = ThingNotifications{
			Actions:    make(map[string]*msg.NotificationMessage),
			Events:     make(map[string]*msg.NotificationMessage),
			Properties: make(map[string]*msg.NotificationMessage),
		}
	}
	switch notif.AffordanceType {
	case msg.AffordanceTypeAction:
		tv.Actions[notif.Name] = notif
	case msg.AffordanceTypeEvent:
		tv.Events[notif.Name] = notif
	case msg.AffordanceTypeProperty:
		tv.Properties[notif.Name] = notif
	}
	store.Things[notif.ThingID] = tv
}

func NewNCacheStore() *VCacheStore {
	store := &VCacheStore{
		Things: make(map[string]ThingNotifications),
	}
	return store
}
