package clientimpl

import (
	"fmt"
	"slices"
	"sync"

	"github.com/hiveot/hivekit/go/api/td"
)

// DirectoryCacheImpl is a concurrent safe local store for thing TDs
// This implements the IDirectoryCache interface.
type DirectoryCacheImpl struct {
	// list of thingIDs in order they have been added
	thingIDs []string
	// link to TD
	cache map[string]*td.TD

	// concurrent safe
	mux sync.RWMutex
}

// Return a TD from the local cache
func (dc *DirectoryCacheImpl) GetThing(thingID string) *td.TD {
	dc.mux.RLock()
	defer dc.mux.RUnlock()
	tdoc, _ := dc.cache[thingID]
	return tdoc
}

// Return a list of TDs from the local cache
func (dc *DirectoryCacheImpl) GetAllThings(offset int, limit int) []*td.TD {
	dc.mux.RLock()
	defer dc.mux.RUnlock()
	var thingID string

	remaining := max(len(dc.thingIDs)-offset, 0)
	size := min(remaining, limit)

	tdList := make([]*td.TD, size)
	for i := range size {
		thingID = dc.thingIDs[offset+i]
		tdList[i] = dc.cache[thingID]
	}
	return tdList
}

// Import the TD into the cache
func (dc *DirectoryCacheImpl) ImportTD(tdoc *td.TD) {
	dc.mux.Lock()
	defer dc.mux.Unlock()

	isNew := dc.cache[tdoc.ID] == nil
	dc.cache[tdoc.ID] = tdoc

	// if the TD is new then add the ThingID list
	if isNew {
		dc.thingIDs = append(dc.thingIDs, tdoc.ID)
	}
}

// Import the TD into the cache
func (dc *DirectoryCacheImpl) ImportTDJson(tdJson string) (*td.TD, error) {
	tdoc, err := td.UnmarshalTD(tdJson)
	if err != nil {
		err = fmt.Errorf("UpdateTD: invalid TD JSON: %s", err.Error())
		return tdoc, err
	}
	dc.ImportTD(tdoc)
	return tdoc, nil
}

// Remove the TD from the cache
func (dc *DirectoryCacheImpl) RemoveTD(thingID string) {
	dc.mux.Lock()
	defer dc.mux.Unlock()
	exists := dc.cache[thingID] != nil
	if !exists {
		return
	}
	delete(dc.cache, thingID)
	// remove the thingID from the list of IDs
	// this is the slow but simple way and rarely used
	for i, id := range dc.thingIDs {
		if id == thingID {
			dc.thingIDs = slices.Delete(dc.thingIDs, i, i)
			return
		}
	}
}

// Create a new instance of the local directory cache
func NewDirectoryCacheImpl() *DirectoryCacheImpl {
	c := &DirectoryCacheImpl{
		thingIDs: make([]string, 0),
		cache:    make(map[string]*td.TD),
	}
	return c
}
