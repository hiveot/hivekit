package vcachemodule

import (
	"sync"
	"time"
)

type CachedValue struct {
	Value     any       `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// ThingValues stores the latest event and property values of a thing
type ThingValues struct {
	Events     map[string]CachedValue `json:"events"`
	Properties map[string]CachedValue `json:"properties"`
}

type VCacheStore struct {
	mux    sync.RWMutex
	Things map[string]ThingValues `json:"things"`
}

// Return the nr of things with cached values
func (store *VCacheStore) GetNrThings() int {
	store.mux.RLock()
	defer store.mux.RUnlock()
	return len(store.Things)
}

// Return the cached value of a property or nil if not found
func (store *VCacheStore) ReadProperty(thingID string, name string) (cv CachedValue, found bool) {
	store.mux.RLock()
	defer store.mux.RUnlock()
	tv, found := store.Things[thingID]
	if found {
		cv, found = tv.Properties[name]
	}
	return cv, found
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

func (store *VCacheStore) WriteProperty(thingID string, name string, value any) {
	store.mux.Lock()
	defer store.mux.Unlock()
	tv, found := store.Things[thingID]
	if !found {
		tv = ThingValues{
			Events:     make(map[string]CachedValue),
			Properties: make(map[string]CachedValue),
		}
	}
	cv := CachedValue{Value: value, Timestamp: time.Now()}
	tv.Properties[name] = cv
	store.Things[thingID] = tv
}

func NewVCacheStore() *VCacheStore {
	store := &VCacheStore{
		Things: make(map[string]ThingValues),
	}
	return store
}
