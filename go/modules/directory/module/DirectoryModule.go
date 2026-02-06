// package module with the directory module factory
package module

import (
	"log/slog"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/modules/bucketstore/module/kvbtree"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/directory/server"
	"github.com/hiveot/hivekit/go/msg"
)

// DirectoryModule is a module for serving a WoT Thing directory.
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
type DirectoryModule struct {
	modules.HiveModuleBase

	// bucket store for use with this module
	bucket      bucketstore.IBucket
	bucketName  string
	bucketStore bucketstore.IBucketStore

	// the RRN messaging API
	msgAPI *server.DirectoryMsgHandler
	// the API servers if enabled
	restAPI *server.DirectoryRestHandler
	// router for rest api
	router *chi.Mux
	// the directory service itself
	// service *service.DirectoryService
	// root directory of the storage area
	storageRoot string
}

// func (m *DirectoryModule) GetService() directory.IDirectoryService {
// 	return m.service
// }

// GetTM returns the module TM document
// It includes forms for messaging access through the WoT.
func (m *DirectoryModule) GetTM() string {
	tmJson := m.msgAPI.GetTM()
	return string(tmJson)
}

// HandleRequest passes the module request messages to the API handler.
func (m *DirectoryModule) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if req.ThingID == m.GetModuleID() {
		err = m.msgAPI.HandleRequest(req, replyTo)
	} else {
		err = m.HiveModuleBase.HandleRequest(req, replyTo)
	}
	return err
}

// Start readies the module for use.
//
// This:
// - opens the bucket store using the configured name.
// - enable the messaging request handler
// - enable the http request handler using the given router
// - updates this service TD in the store
func (m *DirectoryModule) Start(_ string) (err error) {

	moduleID := m.GetModuleID()
	slog.Info("Start: Starting directory module", "moduleID", moduleID)

	storageDir := ""
	if m.storageRoot != "" {
		storageDir = filepath.Join(m.storageRoot, moduleID)
		m.bucketStore = kvbtree.NewKVStore(storageDir)
	} else {
		m.bucketStore = kvbtree.NewKVStore("")
	}
	err = m.bucketStore.Open()
	if err == nil {
		m.bucketName = moduleID
		m.bucket = m.bucketStore.GetBucket(m.bucketName)

		// m.service, err = service.StartDirectoryService(m.bucketStore, bucketName)
	}
	if err == nil {
		m.msgAPI = server.NewDirectoryMsgHandler(moduleID, m)
	}
	if err == nil && m.router != nil {
		m.restAPI = server.StartDirectoryRestHandler(m, m.router)
	}
	return err
}

// Stop any running actions
func (m *DirectoryModule) Stop() {
	slog.Info("Stop: closing directory store")
	err := m.bucket.Close()
	if err != nil {
		slog.Error("Stop: error stopping directory bucket", "err", err.Error())
	}
	m.bucketStore.Close()
}

// Start a new directory server module.
// On start this opens or creates a directory store in root/moduleID.
// Directory entries are stored in the 'directory' bucket.
// If a router is provided this registers the HTTP API with the router.
//
// storageRoot is the root dir of the storage area. Use "" for testing with an in-memory store.
// router is the html server router to register the html API handlers with. nil to ignore.
func NewDirectoryModule(storageRoot string, router *chi.Mux) *DirectoryModule {

	m := &DirectoryModule{
		HiveModuleBase: modules.HiveModuleBase{},
		storageRoot:    storageRoot,
		router:         router,
	}
	m.SetModuleID(directory.DefaultDirectoryThingID)
	var _ modules.IHiveModule = m // interface check

	return m
}
