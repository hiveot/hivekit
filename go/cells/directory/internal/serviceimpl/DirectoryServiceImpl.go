package serviceimpl

import (
	"log/slog"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/bucketstore"
	"github.com/hiveot/hivekit/go/cells/bucketstore/kvbtreestore"
	"github.com/hiveot/hivekit/go/cells/directory"
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
	*cells.HiveCellBase

	// tdBucket store with TD's by thingID
	tdBucket     bucketstore.IBucket
	tdBucketName string
	bucketStore  bucketstore.IBucketStore

	// the http server to expose the TDD on the .well-known/wot path. nil to ignore
	httpServer api.IHttpServer

	// data storage directory
	storageLoc string

	// cache of used TDs and the mutex to access it
	dirTDDJson string
	dirTDD     *td.TD
	tdCache    map[string]*td.TD
	tdCacheMux sync.RWMutex

	// hook to invoke before deleting a TD into the store
	deleteTDHook directory.DeleteTDHook
	// hook to invoke before writing a TD into the store
	writeTDHook directory.WriteTDHook
}

// Return the directory TDD and its json itself
func (svc *DirectoryServiceImpl) GetTDD() (*td.TD, string) {
	return svc.dirTDD, svc.dirTDDJson
}

// Serve reading the directory TDD over http on the well-known path
func (svc *DirectoryServiceImpl) serveReadTDD(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(svc.dirTDDJson))
	// utils.WriteReply(w, true, m.tddJson, nil)
}

// SetTDHooks set the callbacks that are invoked before writing and deleting the TD
// to the directory store.
func (svc *DirectoryServiceImpl) SetTDHooks(
	writeHandler directory.WriteTDHook, deleteHandler directory.DeleteTDHook) {
	svc.deleteTDHook = deleteHandler
	svc.writeTDHook = writeHandler
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

// Start a new thing directory service instance.
// On start this opens or creates a directory store in {home}/{serviceID}.
// Directory entries are stored in the 'directory' bucket.
//
// This:
// - opens the bucket store using the thingID as the bucket name.
// - enable the messaging request handler
// - enable the http request handler using the given router
// - include the directory TDD itself in the store
//
// The directory publishes a TD that describes how it can be reached. This TD needs
// to include the security details and forms, which are transport specific.
//
// To expose the http API create the DirectoryHttpHandler provide it here.
// Optionally include the list of other transport.
//
//	thingID is the instance ID of the directory server or "" for default
//	storageDir is the directory where the service stores its data. Use "" for testing with an in-memory store.
//	httpServer is used to expose the directory TDD on the well-known path.
//	transports is a list of transports that should be included in the TDD security and forms. nil to not include these.
func StartDirectoryServiceImpl(
	thingID string, storageDir string, httpServer api.IHttpServer,
	transports []api.ITransportServer) (*DirectoryServiceImpl, error) {

	slog.Info("Start: Starting directory service")

	if thingID == "" {
		thingID = directory.DefaultDirectoryThingID
	}

	// Use the transports to generate a tdd from the tm
	// option 2: use transport of sender
	tm := string(directory.DirectoryTMJson)
	dirTDD, _ := td.UnmarshalTD(tm)
	if thingID != "" {
		dirTDD.ID = thingID
	}
	// add the forms for additional endpoints
	// if len(transports) > 0 {
	for _, tp := range transports {
		if tp == nil {
			slog.Error("NewDirectoryService: Transports has a nil transport")
		} else {
			tp.AddTDSecForms(dirTDD, true)
		}
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
		HiveCellBase: cells.NewHiveCellBase(thingID, 0),
		bucketStore:  bucketStore,
		httpServer:   httpServer,
		storageLoc:   storageDir,
		tdBucket:     tdBucket,
		dirTDD:       dirTDD,
		dirTDDJson:   td.MarshalTD(dirTDD),
		tdCache:      make(map[string]*td.TD),
	}

	// service the directory TDD on the well-known path
	if httpServer != nil {
		protRoute := httpServer.GetProtectedRoute()
		protRoute.Get(directory.WellKnownWoTPath, svc.serveReadTDD)
	}

	var _ directory.IDirectoryService = svc // interface check

	return svc, err
}
