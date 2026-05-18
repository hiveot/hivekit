package internal

import (
	"log/slog"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketstorepkg "github.com/hiveot/hivekit/go/modules/bucketstore/pkg"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// DirectoryServer is a module for serving a WoT Thing directory.
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
type DirectoryServer struct {
	modules.HiveModuleBase
	// The thingID this service identifies as
	directoryThingID string

	// tdBucket store with TD's by thingID
	tdBucket     bucketstore.IBucket
	tdBucketName string
	bucketStore  bucketstore.IBucketStorage

	// the RRN messaging API for the directory itself
	msgAPI *DirectoryMsgHandler

	// the http server to expose the TDD on the .well-known/wot path. nil to ignore
	httpServer transports.IHttpServer

	// data storage directory
	storageLoc string

	// cache of used TDs and the mutex to access it
	tddJson    string
	tdCache    map[string]*td.TD
	tdCacheMux sync.RWMutex

	// hook to invoke before deleting a TD into the store
	deleteTDHook directory.DeleteTDHook
	// hook to invoke before writing a TD into the store
	writeTDHook directory.WriteTDHook
}

// GetAgentInfo provides information on Things registered by an agent
func (m *DirectoryServer) GetAgentInfo(agentID string) (
	info directory.AgentInfo, found bool) {

	// how are agents tracked?
	// option 1: separate bucket with all agents
	// option 2: a bucket per agent
	// option 3: agent as prefix of thingID
	//   + allows seek/filter while iterating
	//   - cant lookup by thingID

	return info, false
}

// HandleRequest passes the module request messages to the API handler.
func (m *DirectoryServer) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if req.ThingID == m.directoryThingID {
		err = m.msgAPI.HandleRequest(req, replyTo)
	} else {
		err = m.HiveModuleBase.HandleRequest(req, replyTo)
	}
	return err
}

// Serve reading the directory TDD over http on the well-known path
func (m *DirectoryServer) serveReadTDD(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(m.tddJson))
	// utils.WriteReply(w, true, m.tddJson, nil)
}

// SetTDHooks set the callbacks that are invoked before writing and deleting the TD
// to the directory store.
func (m *DirectoryServer) SetTDHooks(
	writeHandler directory.WriteTDHook, deleteHandler directory.DeleteTDHook) {
	m.deleteTDHook = deleteHandler
	m.writeTDHook = writeHandler
}

// Start readies the module for use.
//
// This:
// - opens the bucket store using the configured name.
// - enable the messaging request handler
// - enable the http request handler using the given router
// - updates this service TD in the store
func (m *DirectoryServer) Start() (err error) {

	storagePath := m.storageLoc
	slog.Info("Start: Starting directory module")

	// if no storageLoc is set, use the in-memory store
	if m.storageLoc != "" {
		storagePath = filepath.Join(m.storageLoc, m.directoryThingID+".kvbtree")
	}
	m.bucketStore, err = bucketstorepkg.NewBucketStore(storagePath, bucketstore.BackendKVBTree)

	err = m.bucketStore.Open()
	if err == nil {
		m.tdBucketName = m.directoryThingID
		m.tdBucket = m.bucketStore.GetBucket(m.tdBucketName)
	}
	if err == nil {
		m.msgAPI = NewDirectoryMsgHandler(m.directoryThingID, m)
	}

	if m.httpServer != nil {
		protRoute := m.httpServer.GetProtectedRoute()
		protRoute.Get(directory.WellKnownWoTPath, m.serveReadTDD)
	}

	return err
}

// Stop any running actions
func (m *DirectoryServer) Stop() {
	slog.Info("Stop: Stopping directory module")
	err := m.tdBucket.Close()
	if err != nil {
		slog.Error("Stop: error stopping directory bucket", "err", err.Error())
	}
	m.bucketStore.Close()
}

// Start a new thing directory server module.
// On start this opens or creates a directory store in root/moduleID.
// Directory entries are stored in the 'directory' bucket.
//
// The directory publishes a TD that describes how it can be reached. This TD needs
// to include the security details and forms, which are transport specific.
//
// To expose the http API create the DirectoryHttpHandler module provide it here.
// Optionally include the list of other transports.
//
//	thingID is the instance ID of the directory server.
//	location is the location where the module stores its data. Use "" for testing with an in-memory store.
//	httpServer is used to expose the directory TDD on the well-known path.
//	transports is a list of transports that should be included in the TDD security and forms. nil to not include these.
func NewDirectoryServer(
	thingID string, location string, httpServer transports.IHttpServer,
	transports []transports.ITransportServer) *DirectoryServer {

	if thingID == "" {
		thingID = directory.DefaultDirectoryThingID
	}

	// Use the transports to generate a tdd from the tm
	// option 2: use transport of sender
	tm := string(directory.DirectoryTMJson)
	tddDoc, _ := td.UnmarshalTD(tm)
	tddDoc.ID = thingID
	// add the forms for additional endpoints
	if len(transports) > 0 {
		for _, tp := range transports {
			tp.AddTDSecForms(tddDoc, false) // tbd
		}
	}
	tddJson, _ := td.MarshalTD(tddDoc)
	m := &DirectoryServer{
		HiveModuleBase:   modules.HiveModuleBase{},
		directoryThingID: thingID,
		httpServer:       httpServer,
		storageLoc:       location,
		tddJson:          tddJson,
		tdCache:          make(map[string]*td.TD),
	}

	var _ directory.IDirectoryService = m // interface check

	return m
}
