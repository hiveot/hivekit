// package directoryserver with the directory server module
package directoryserver

import (
	"log/slog"
	"path/filepath"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketstoreapi "github.com/hiveot/hivekit/go/modules/bucketstore/api"
	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
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

	// tdBucket store with TD's by thingID
	tdBucket     bucketstoreapi.IBucket
	tdBucketName string
	bucketStore  bucketstoreapi.IBucketStore

	// http server serving the REST API
	httpServer transports.IHttpServer
	// the RRN messaging API for the directory itself
	msgAPI *DirectoryMsgHandler
	// the API servers if enabled
	restAPI *DirectoryRestHandler
	// root directory of the storage area
	storageRoot string

	// hook to invoke before writing a TD into the store
	writeTDHook directoryapi.WriteTDHook
	// hook to invoke before deleting a TD into the store
	deleteTDHook directoryapi.DeleteTDHook
}

// GetAgentInfo provides information on Things registered by an agent
func (m *DirectoryServer) GetAgentInfo(agentID string) (
	info directoryapi.AgentInfo, found bool) {

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
	if req.ThingID == m.GetModuleID() {
		err = m.msgAPI.HandleRequest(req, replyTo)
	} else {
		err = m.HiveModuleBase.HandleRequest(req, replyTo)
	}
	return err
}

// SetTDHooks set the callbacks that are invoked before writing and deleting the TD
// to the directory store.
func (m *DirectoryServer) SetTDHooks(
	writeHandler directoryapi.WriteTDHook, deleteHandler directoryapi.DeleteTDHook) {
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
func (m *DirectoryServer) Start(_ string) (err error) {

	moduleID := m.GetModuleID()
	slog.Info("Start: Starting directory module", "moduleID", moduleID)

	storageDir := ""
	if m.storageRoot != "" {
		storageDir = filepath.Join(m.storageRoot, moduleID)
	}
	m.bucketStore, err = bucketstore.NewBucketStore(storageDir, bucketstoreapi.BackendKVBTree)

	err = m.bucketStore.Open()
	if err == nil {
		m.tdBucketName = moduleID
		m.tdBucket = m.bucketStore.GetBucket(m.tdBucketName)
	}
	if err == nil {
		m.msgAPI = NewDirectoryMsgHandler(moduleID, m)
	}
	if err == nil && m.httpServer != nil {
		m.restAPI = StartDirectoryRestHandler(m, m.httpServer)
	}
	return err
}

// Stop any running actions
func (m *DirectoryServer) Stop() {
	slog.Info("Stop: closing directory store")
	err := m.tdBucket.Close()
	if err != nil {
		slog.Error("Stop: error stopping directory bucket", "err", err.Error())
	}
	m.bucketStore.Close()
}

// Start a new directory server module.
// On start this opens or creates a directory store in root/moduleID.
// Directory entries are stored in the 'directory' bucket.
//
// If a http server is provided this registers the HTTP API with the router and serves
// its TD on the .well-known/wot endpoint as per discovery specification.
//
// storageRoot is the root dir of the storage area. Use "" for testing with an in-memory store.
// router is the html server router to register the html API handlers with. nil to ignore.
func NewDirectoryServer(storageRoot string, httpServer transports.IHttpServer) *DirectoryServer {

	m := &DirectoryServer{
		HiveModuleBase: modules.HiveModuleBase{},
		storageRoot:    storageRoot,
		httpServer:     httpServer,
	}
	m.SetModuleID(directoryapi.DefaultDirectoryServiceID)
	if httpServer == nil {
		slog.Warn("NewDirectoryModule: no httpServer provided. HTTP interface not active.")
	}
	var _ directoryapi.IDirectoryServer = m // interface check

	return m
}
