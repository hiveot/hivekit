// Package internal with service methods
package service

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/directory"
)

// CreateThing adds or replaces the TD in the store.
func (svc *DirectoryServiceImpl) CreateThing(agentID string, tdJson string) error {

	// TODO: link the TD to the agent that created it

	return svc.UpdateThing(agentID, tdJson)
}

// DeleteThing removes a Thing TD document from the store and send a notification
func (svc *DirectoryServiceImpl) DeleteThing(agentID string, thingID string) (err error) {

	// TODO: check that the agentID is linked to this TD, or an administrator.

	slog.Info("Delete Thing",
		slog.String("agentID", agentID), slog.String("thingID", thingID))

	// The hook can cancel the write
	if svc.deleteTDHook != nil {
		err = svc.deleteTDHook(agentID, thingID)
	}
	if err == nil {
		err = svc.tdBucket.Delete(thingID)
		svc.tdCacheMux.Lock()
		delete(svc.tdCache, thingID)
		svc.tdCacheMux.Unlock()

		notif := msg.NewNotificationMessage(svc.GetThingID(), msg.AffordanceTypeEvent,
			svc.GetThingID(), directory.ThingDeletedEvent, thingID)
		svc.ForwardNotification(notif)
	}
	return err
}

// Return an instance of the thing TD if avaialable.
// These instances are cached so successive requests are efficient.
func (svc *DirectoryServiceImpl) GetTD(thingID string) *td.TD {
	svc.tdCacheMux.RLock()
	tdoc, found := svc.tdCache[thingID]
	if found {
		return tdoc
	}
	svc.tdCacheMux.RUnlock()
	tdJSON, err := svc.RetrieveThing(thingID)
	if err != nil {
		return nil
	}
	tdoc, err = td.UnmarshalTD(tdJSON)
	if err == nil {
		svc.tdCacheMux.Lock()
		svc.tdCache[thingID] = tdoc
		svc.tdCacheMux.Unlock()
	}
	return tdoc
}

//func (svc *DirectoryService) QueryThings(
//	senderID string, args digitwin.DirectoryQueryTDsArgs) (tdDocuments []string, err error) {
//	//svc.DtwStore.QueryDTDs(args)
//	return nil, fmt.Errorf("Not yet implemented")
//}

// RetrieveAllThings returns a batch of TD documents
// This returns a list of JSON encoded digital twin TD documents
func (svc *DirectoryServiceImpl) RetrieveAllThings(offset int, limit int) (tdList []string, err error) {
	tdList = make([]string, 0)

	cursor, err := svc.tdBucket.Cursor()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = directory.DefaultLimit
	}
	itemsToRead := limit
	if offset != 0 {
		_ = cursor.Skip(offset)
	}

	for {
		// read in batches of defaultLimit TD documents
		readCount := min(directory.DefaultLimit, itemsToRead)
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

// RetrieveThing returns a JSON encoded TD document
func (svc *DirectoryServiceImpl) RetrieveThing(thingID string) (tdJSON string, err error) {
	tdBytes, err := svc.tdBucket.Get(thingID)
	tdJSON = string(tdBytes)
	return tdJSON, err
}

// Stop the service and close the storage bucket
// The bucketStore itself is not closed on Stop.
// func (svc *DirectoryModule) Stop() error {
// 	err := svc.bucket.Close()
// 	return err
// }

// UpdateThing replaces the TD in the store.
// If the thing doesn't exist in the store it is added.
func (svc *DirectoryServiceImpl) UpdateThing(agentID string, tdJson string) error {

	// TODO: verify that the TD is the agent that created it.

	// validate the TD
	tdi, err := td.UnmarshalTD(tdJson)
	if err != nil {
		slog.Error("UpdateThing. Error unmarshalling TD",
			slog.String("agentID", agentID), "err", err.Error())
		return err
	}
	slog.Info("UpdateThing",
		slog.String("agentID", agentID), slog.String("thingID", tdi.ID))

	// the agentID is stored to determine where to route requests to, when using RC agents
	// RC agents have no 'base' address.
	// tdi.AgentID = agentID

	// The hook can modify the TD or cancel the write
	if svc.writeTDHook != nil {
		tdi2, err := svc.writeTDHook(agentID, tdi)
		if err != nil {
			return err
		} else if tdi2 == nil {
			slog.Error("UpdateThing. writeTDHook returns a nil TD", "thingID", tdi.ID)
			return fmt.Errorf("UpdateThing: Internal error, the writeTDHook returns a nil TD")
		}
		// replace the TD with the one provided by the hook
		tdJson = td.MarshalTD(tdi2)
	}

	err = svc.tdBucket.Set(tdi.ID, []byte(tdJson))
	// reload the td instance next time someone asks
	svc.tdCacheMux.Lock()
	delete(svc.tdCache, tdi.ID)
	svc.tdCacheMux.Unlock()

	notif := msg.NewNotificationMessage(svc.GetThingID(), msg.AffordanceTypeEvent,
		svc.GetThingID(), directory.ThingUpdatedEvent, tdJson)
	svc.ForwardNotification(notif)

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
