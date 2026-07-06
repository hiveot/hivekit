package service

import (
	"log/slog"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketstorepkg "github.com/hiveot/hivekit/go/modules/bucketstore/pkg"
	"github.com/hiveot/hivekit/go/modules/directory"
)

// DirectoryServiceImpl is a module for serving a WoT Thing directory.
// This implements the IHiveModule and IDirectoryService interfaces.
//
// The directory can be accessed:
//  1. Natively from golang. The module supports the IDirectoryService interface.
//  2. Using hivekit RRN messaging (request-response-notification). See DirectoryMsgHandler.go
//  3. Using the HTTP REST API as per WoT specification. See DirectoryRestHandler.go.
//
// See directory-tm.json for the WoT TM definition of the module.
//
// The module is configured using yaml.
//
// This uses the fast and lightweight kvbtree bucket store to persist TD documents.
type DirectoryServiceImpl struct {
	*modules.HiveModuleBase

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

// Start readies the module for use.
//
// This:
// - opens the bucket store using the configured name.
// - enable the messaging request handler
// - enable the http request handler using the given router
// - updates the directory TDD in the store
func (svc *DirectoryServiceImpl) Start() (err error) {

	storagePath := svc.storageLoc
	thingID := svc.GetThingID()
	slog.Info("Start: Starting directory module")

	// if no storageLoc is set, use the in-memory store
	if svc.storageLoc != "" {
		storagePath = filepath.Join(svc.storageLoc, thingID+".kvbtree")
	}
	svc.bucketStore, err = bucketstorepkg.NewBucketStore(storagePath, bucketstore.BackendKVBTree)

	err = svc.bucketStore.Open()
	if err == nil {
		svc.tdBucketName = thingID
		svc.tdBucket = svc.bucketStore.GetBucket(svc.tdBucketName)
	}

	if svc.httpServer != nil {
		protRoute := svc.httpServer.GetProtectedRoute()
		protRoute.Get(directory.WellKnownWoTPath, svc.serveReadTDD)
	}

	return err
}

// Stop any running actions
func (svc *DirectoryServiceImpl) Stop() {
	slog.Info("Stop: Stopping directory module")
	err := svc.tdBucket.Close()
	if err != nil {
		slog.Error("Stop: error stopping directory bucket", "err", err.Error())
	}
	svc.bucketStore.Close()
}

// Start a new thing directory service module.
// On start this opens or creates a directory store in {home}/{moduleID}.
// Directory entries are stored in the 'directory' bucket.
//
// The directory publishes a TD that describes how it can be reached. This TD needs
// to include the security details and forms, which are transport specific.
//
// To expose the http API create the DirectoryHttpHandler module provide it here.
// Optionally include the list of other transport.
//
//	thingID is the instance ID of the directory server or "" for default
//	location is the location where the module stores its data. Use "" for testing with an in-memory store.
//	httpServer is used to expose the directory TDD on the well-known path.
//	transports is a list of transports that should be included in the TDD security and forms. nil to not include these.
func NewDirectoryServiceImpl(
	thingID string, location string, httpServer api.IHttpServer,
	transports []api.ITransportServer) *DirectoryServiceImpl {

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
	// }
	tddJson := td.MarshalTD(dirTDD)
	m := &DirectoryServiceImpl{
		HiveModuleBase: modules.NewHiveModuleBase(thingID, 0),
		httpServer:     httpServer,
		storageLoc:     location,
		dirTDD:         dirTDD,
		dirTDDJson:     tddJson,
		tdCache:        make(map[string]*td.TD),
	}

	var _ directory.IDirectoryService = m // interface check

	return m
}
