// Package directoryserver with service methods
package module

import (
	"fmt"
	"log/slog"

	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
	"github.com/hiveot/hivekit/go/wot/td"
)

// CreateThing adds or replaces the TD in the store.
func (m *DirectoryModuleServer) CreateThing(agentID string, tdJson string) error {

	// TODO: link the TD to the agent that created it

	return m.UpdateThing(agentID, tdJson)
}

// DeleteThing removes a Thing TD document from the store.
func (m *DirectoryModuleServer) DeleteThing(agentID string, thingID string) (err error) {

	// TODO: check that the agentID is linked to this TD, or an administrator.

	slog.Info("Delete Thing",
		slog.String("agentID", agentID), slog.String("thingID", thingID))

	// The hook can cancel the write
	if m.deleteTDHook != nil {
		err = m.deleteTDHook(agentID, thingID)
	}
	if err == nil {
		err = m.tdBucket.Delete(thingID)
		m.tdCacheMux.Lock()
		delete(m.tdCache, thingID)
		m.tdCacheMux.Unlock()
	}
	return err
}

// Return an instance of the thing TD if avaialable.
// These instances are cached so successive requests are efficient.
func (m *DirectoryModuleServer) GetTD(thingID string) *td.TD {
	m.tdCacheMux.RLock()
	tdi, found := m.tdCache[thingID]
	if found {
		return tdi
	}
	m.tdCacheMux.RUnlock()
	tdJSON, err := m.RetrieveThing(thingID)
	if err != nil {
		return nil
	}
	tdi, err = td.UnmarshalTD(tdJSON)
	if err == nil {
		m.tdCacheMux.Lock()
		m.tdCache[thingID] = tdi
		m.tdCacheMux.RUnlock()
	}
	return tdi
}

//func (svc *DirectoryService) QueryThings(
//	senderID string, args digitwin.DirectoryQueryTDsArgs) (tdDocuments []string, err error) {
//	//svc.DtwStore.QueryDTDs(args)
//	return nil, fmt.Errorf("Not yet implemented")
//}

// RetrieveThing returns a JSON encoded TD document
func (m *DirectoryModuleServer) RetrieveThing(thingID string) (tdJSON string, err error) {
	tdBytes, err := m.tdBucket.Get(thingID)
	tdJSON = string(tdBytes)
	return tdJSON, err
}

// RetrieveAllThings returns a batch of TD documents
// This returns a list of JSON encoded digital twin TD documents
func (m *DirectoryModuleServer) RetrieveAllThings(offset int, limit int) (tdList []string, err error) {
	tdList = make([]string, 0)

	cursor, err := m.tdBucket.Cursor()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = directoryapi.DefaultLimit
	}
	itemsToRead := limit
	if offset != 0 {
		_ = cursor.Skip(offset)
	}

	for {
		// read in batches of defaultLimit TD documents
		readCount := min(directoryapi.DefaultLimit, itemsToRead)
		itemsToRead -= readCount
		tdmap, itemsRemaining := cursor.NextN(uint(readCount))
		for _, tdBin := range tdmap {
			tdList = append(tdList, string(tdBin))
		}
		if !itemsRemaining || itemsToRead <= 0 {
			break
		}
	}
	return tdList, err
}

// Stop the service and close the storage bucket
// The bucketStore itself is not closed on Stop.
// func (svc *DirectoryModule) Stop() error {
// 	err := svc.bucket.Close()
// 	return err
// }

// UpdateThing replaces the TD in the store.
// If the thing doesn't exist in the store it is added.
func (m *DirectoryModuleServer) UpdateThing(agentID string, tdJson string) error {

	// TODO: verify that the TD is the agent that created it.

	// validate the TD
	tdi, err := td.UnmarshalTD(tdJson)
	if err != nil {
		slog.Info("UpdateThing. Error unmarshalling TD",
			slog.String("agentID", agentID), "err", err.Error())
		return err
	}
	slog.Info("UpdateThing",
		slog.String("agentID", agentID), slog.String("thingID", tdi.ID))

	// The hook can modify the TD or cancel the write
	if m.writeTDHook != nil {
		tdi2, err := m.writeTDHook(agentID, tdi)
		if err != nil {
			return err
		} else if tdi2 == nil {
			slog.Error("UpdateThing. writeTDHook returns a nil TD", "thingID", tdi.ID)
			return fmt.Errorf("UpdateThing: Internal error, the writeTDHook returns a nil TD")
		}
		tdJson, _ = td.MarshalTD(tdi2)
	}

	err = m.tdBucket.Set(tdi.ID, []byte(tdJson))
	// reload the td instance next time someone asks
	m.tdCacheMux.Lock()
	delete(m.tdCache, tdi.ID)
	m.tdCacheMux.Unlock()

	return err

}

// StartDirectoryService creates a new instance of the directory service and open
// the storage bucket for use. Call Stop() to close the bucket(s) when done.
// The bucketStore itself is not closed on Stop.
//
// bucketStore is the store to open/create the directory bucket into.
// bucketName is the name of the bucket to use. Default is "directory".
// func StartDirectoryService(bucketStore bucketstore.IBucketStore, bucketName string) (*DirectoryService, error) {
// 	if bucketName == "" {
// 		bucketName = "directory"
// 	}
// 	dirSrv := &DirectoryService{
// 		bucket:     bucketStore.GetBucket(bucketName),
// 		bucketName: bucketName,
// 	}

// 	return dirSrv, nil
// }
