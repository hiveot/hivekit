package ncachemodule

import (
	"sync"

	"github.com/hiveot/hivekit/go/msg"
)

// ThingNotifications stores the latest event and property values of a thing
type ThingNotifications struct {
	Events     map[string]*msg.NotificationMessage `json:"events"`
	Properties map[string]*msg.NotificationMessage `json:"properties"`
}

type NCacheStore struct {
	mux    sync.RWMutex
	Things map[string]ThingNotifications `json:"things"`
}

// Return the nr of things with cached values
func (store *NCacheStore) GetNrThings() int {
	store.mux.RLock()
	defer store.mux.RUnlock()
	return len(store.Things)
}

// Return the latest cached event or nil if not found
func (store *NCacheStore) ReadEvent(thingID string, name string) (event *msg.NotificationMessage) {
	store.mux.RLock()
	defer store.mux.RUnlock()
	tn, found := store.Things[thingID]
	if found {
		event, found = tn.Events[name]
	}
	return event
}

// Return the cached value of a property or nil if not found
func (store *NCacheStore) ReadProperty(thingID string, name string) (prop *msg.NotificationMessage) {
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
func (store *NCacheStore) ReadMultipleProperties(
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
func (store *NCacheStore) RemoveProperty(thingID string, name string) {
	store.mux.Lock()
	defer store.mux.Unlock()
	tv, found := store.Things[thingID]
	if found {
		delete(tv.Properties, name)
	}
}

func (store *NCacheStore) WriteEvent(notif *msg.NotificationMessage) {
	store.mux.Lock()
	defer store.mux.Unlock()
	tv, found := store.Things[notif.ThingID]
	if !found {
		tv = ThingNotifications{
			Events:     make(map[string]*msg.NotificationMessage),
			Properties: make(map[string]*msg.NotificationMessage),
		}
	}
	tv.Events[notif.Name] = notif
	store.Things[notif.ThingID] = tv
}

func (store *NCacheStore) WriteProperty(notif *msg.NotificationMessage) {
	store.mux.Lock()
	defer store.mux.Unlock()
	tv, found := store.Things[notif.ThingID]
	if !found {
		tv = ThingNotifications{
			Events:     make(map[string]*msg.NotificationMessage),
			Properties: make(map[string]*msg.NotificationMessage),
		}
	}
	tv.Properties[notif.Name] = notif
	store.Things[notif.ThingID] = tv
}

func NewNCacheStore() *NCacheStore {
	store := &NCacheStore{
		Things: make(map[string]ThingNotifications),
	}
	return store
}
