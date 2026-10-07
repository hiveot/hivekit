package internal

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/bucketstore"
	"github.com/hiveot/hivekit/go/cells/bucketstore/kvbtreestore"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/thing"
)

// DirectoryServiceImpl serves a WoT Thing directory.
// This implements the IHiveCell and IDirectoryService interfaces.
//
// The directory can be accessed:
//  1. Natively from golang. The service supports the IDirectoryService interface.
//  2. Using hivekit RRN messaging (request-response-notification). See DirectoryMsgHandler.go
//  3. Using the HTTP REST API as per WoT specification. See DirectoryRestHandler.go.
//
// See directory-tm.json for the WoT TM definition of the service.
//
// The service is configured using yaml.
//
// This uses the fast and lightweight kvbtree bucket store to persist TD documents.
type DirectoryServiceImpl struct {
	// *cells.HiveCellBase
	*thing.ExposedThing

	// tdBucket store with TD's by thingID
	tdBucket     bucketstore.IBucket
	tdBucketName string
	bucketStore  bucketstore.IBucketStore

	// data storage directory
	storageLoc string

	// the directory TDD
	dirTDD *td.TD

	// cache of used TDs and the mutex to access it
	tdCache    map[string]*td.TD
	tdCacheMux sync.RWMutex

	// optional hook to add forms for when TDs dont have any.
	addFormsHook func(*td.TD)

	// optional hook to invoke before deleting a TD into the store
	deleteTDHook directory.DeleteTDHook
	// optional hook to invoke before writing a TD into the store
	writeTDHook directory.WriteTDHook
}

// CreateThing adds or replaces the TD in the store.
func (svc *DirectoryServiceImpl) CreateThing(senderID string, tdJson string) error {

	return svc.UpdateThing(senderID, tdJson)
}

// DeleteThing removes a Thing TD document from the store and send a notification
func (svc *DirectoryServiceImpl) DeleteThing(senderID string, thingID string) (err error) {

	// TODO: check that the senderID is linked to this TD, or an administrator.

	slog.Info("Delete Thing",
		slog.String("senderID", senderID), slog.String("thingID", thingID))

	// The hook can cancel the write
	if svc.deleteTDHook != nil {
		err = svc.deleteTDHook(senderID, thingID)
	}
	if err == nil {
		err = svc.tdBucket.Delete(thingID)
		svc.tdCacheMux.Lock()
		delete(svc.tdCache, thingID)
		svc.tdCacheMux.Unlock()

		svc.PubEvent(svc.GetID(), directory.ThingDeletedEvent, thingID)
	}
	return err
}

// Return an instance of the thing TD if available.
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

// Return the directory TDD and its json itself
// This has forms included for the servers provided during creation.
func (svc *DirectoryServiceImpl) GetTDD() *td.TD {
	return svc.dirTDD
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

// SetTDHooks set the callbacks that are invoked before writing and deleting the TD
// to the directory store.
func (svc *DirectoryServiceImpl) SetTDHooks(
	writeHandler directory.WriteTDHook, deleteHandler directory.DeleteTDHook) {
	svc.deleteTDHook = deleteHandler
	svc.writeTDHook = writeHandler
}

// Start publishes the directory TD
func (svc *DirectoryServiceImpl) Start() {
	slog.Info("Starting DirectoryService", "ThingID", svc.GetID())
	// initialize the nr records prop
	bucketInfo := svc.tdBucket.Info()
	svc.PubProperty(svc.GetID(), directory.PropNrThings, bucketInfo.NrRecords, true)

}

// Stop any running actions
func (svc *DirectoryServiceImpl) Stop() {
	slog.Info("Stop: Stopping directory service")
	err := svc.tdBucket.Close()
	if err != nil {
		slog.Error("Stop: error stopping directory bucket", "err", err.Error())
	}
	svc.bucketStore.Close()
}

// UpdateThing replaces the TD in the store.
// If the thing doesn't exist in the store it is added.
//
// If the given TD has no forms then it is either a local service (no senderID),
// or an RC device. In both cases forms need to be added to ensure the device is
// reachable. If this is the TD of an RC device then set the 'RCID' of the TD to
// the senderID.
//
// This invokes the writeTDHook to support updating the TD before it is added
// to the directory. Intended for digital twin, validation and filtering use-cases.
//
//	senderID is the authenticated clientID updating the TD. "" when local.
//	tdJSON contains the TD to update in the directory store.
func (svc *DirectoryServiceImpl) UpdateThing(senderID string, tdJSON string) error {

	// validate the TD
	tdoc, err := td.UnmarshalTD(tdJSON)
	if err != nil {
		slog.Error("UpdateThing. Error unmarshalling TD",
			slog.String("senderID", senderID), "err", err.Error())
		return err
	}
	if tdoc.ID == "" {
		err = fmt.Errorf("TD is missing an ID. Title='%s'", tdoc.Title)
		slog.Error("UpdateThing. TD is missing an ID",
			slog.String("senderID", senderID), "title", tdoc.Title, "err", err.Error())
		return err
	}
	slog.Info("UpdateThing",
		slog.String("senderID", senderID), slog.String("thingID", tdoc.ID))

	// if the document has no forms this is either a local service or an RC device.
	if tdoc.Base == "" && len(tdoc.Forms) == 0 {
		if svc.addFormsHook != nil {
			svc.addFormsHook(tdoc)
		}
		// A missing senderID means the update was submitted locally, not via a server.
		if senderID != "" {
			// this is an RC connected device that can be reached via its connection
			tdoc.SetRCID(senderID)
		}
	}
	// TODO: verify ownership of TDs
	// get the old TD to verify the sender is the same
	// oldTDJSON, err := svc.tdBucket.Get(tdoc.ID)
	// if err == nil {
	// 	oldTD, err2 := td.UnmarshalTD(string(oldTDJSON))
	// 	if err2 == nil {
	// 		oldRCID := oldTD.GetRCID()
	// 		if oldRCID != "" && oldRCID != senderID {
	// 			err := fmt.Errorf("UpdateThing: Sender '%s' for TD '%s' is not the owner", senderID, tdoc.ID)
	// 			slog.Warn(err.Error())
	// 			return err
	// 		}
	// 	}
	// }

	// The hook can modify the TD or cancel the write
	if svc.writeTDHook != nil {
		tdi2, err := svc.writeTDHook(senderID, tdoc)
		if err != nil {
			return err
		} else if tdi2 == nil {
			slog.Error("UpdateThing. writeTDHook returns a nil TD", "thingID", tdoc.ID)
			return fmt.Errorf("UpdateThing: Internal error, the writeTDHook returns a nil TD")
		}
		// replace the TD with the one provided by the hook
		tdJSON = td.MarshalTD(tdi2)
	} else {
		// the td was updated with the senderID
		// do not update the 'Modified' time as this update is not made
		// by the device.
		tdJSON = td.MarshalTD(tdoc)
	}

	err = svc.tdBucket.Set(tdoc.ID, []byte(tdJSON))
	// update the cached td instance as well
	svc.tdCacheMux.Lock()
	svc.tdCache[tdoc.ID] = tdoc
	svc.tdCacheMux.Unlock()
	svc.PubEvent(svc.GetID(), directory.ThingUpdatedEvent, tdJSON)

	// update the nr records prop
	bucketInfo := svc.tdBucket.Info()
	svc.PubProperty(svc.GetID(), directory.PropNrThings, bucketInfo.NrRecords, true)

	return err

}

// Create a ready-to-use thing directory service instance.
//
// Directory entries are stored in the 'directory' bucket.
//
// This:
// - opens the bucket store using the thingID as the bucket name.
// - enable the TDD download handler using the given http server
// - include the directory TDD itself in the store
//
// The directory publishes a TD that describes how it can be reached. This TD needs
// to include the security details and forms, which are transport specific.
//
// To expose the http API create the DirectoryHttpHandler provide it here.
// Optionally include the list of other transport.
//
// To modify the TDD with available transports, provide the getTransports callback.
//
//	thingID is the instance ID of the directory server or "" for the default {host}:directory
//	storageDir is the directory where the service stores its data. Use "" for testing with an in-memory store.
//	addForms optional hook to add forms to TDs that dont have any.
func NewDirectoryServiceImpl(
	thingID string, storageDir string, addForms func(*td.TD),
) (*DirectoryServiceImpl, error) {

	slog.Info("NewDirectoryServiceImpl running the directory service")

	if thingID == "" {
		thingID = directory.DirectoryServiceDefaultThingID
	}

	// create the directory TD from the json file
	tdoc := string(directory.DirectoryTDJson)
	dirTDD, _ := td.UnmarshalTD(tdoc)
	dirTDD.ID = thingID
	dirTDD.SetType(directory.DirectoryServiceCellType)
	if addForms != nil {
		addForms(dirTDD)
	}

	// if a storageDir is set use the thingID as filename. Otherwise use the in-memory store
	storageFile := ""
	if storageDir != "" {
		storageFile = filepath.Join(storageDir, thingID+".kvbtree")
	}
	bucketStore, err := kvbtreestore.OpenKVBTreeStore(storageFile)
	if err != nil {
		return nil, err
	}
	tdBucket := bucketStore.GetBucket(thingID)

	svc := &DirectoryServiceImpl{
		ExposedThing: thing.NewExposedThing(thingID, nil),
		bucketStore:  bucketStore,
		dirTDD:       dirTDD,
		addFormsHook: addForms,
		storageLoc:   storageDir,
		tdBucket:     tdBucket,
		tdCache:      make(map[string]*td.TD),
	}

	var _ directory.IDirectoryService = svc // interface check

	return svc, err
}
