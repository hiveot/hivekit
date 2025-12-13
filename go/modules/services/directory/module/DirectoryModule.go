// package module with the directory module factory
package module

import (
	_ "embed"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/messaging"
	"github.com/hiveot/hivekit/go/modules/services/bucketstore"
	"github.com/hiveot/hivekit/go/modules/services/bucketstore/kvbtree"
	"github.com/hiveot/hivekit/go/modules/services/directory"
	"github.com/hiveot/hivekit/go/modules/services/directory/api"
	"github.com/hiveot/hivekit/go/modules/services/directory/service"
)

// const bucketName = "directory"

// DirectoryModule is a module for accessing a WoT Thing directory.
//
// The module is configured using yaml.
// Any database supporting the IBucketStore interface can be used as underlying storage.
// A REST API is supported for accessing the directory over HTTP as per WoT specification
// using the endpoints defined in the TD. (directory.json)
type DirectoryModule struct {
	modules.HiveModuleBase

	// bucket store for use with this module
	bucketStore bucketstore.IBucketStore
	// the WoT messaging API
	msgAPI *api.DirectoryMsgHandler
	// the API servers if enabled
	restAPI *api.DirectoryRestHandler
	// router for rest api
	router *chi.Mux
	// the directory service itself
	service *service.DirectoryService
	// root directory of the storage area
	storageRoot string
}

func (m *DirectoryModule) GetService() directory.IDirectory {
	return m.service
}

// HandleRequest passes the module request messages to the API handler.
func (m *DirectoryModule) HandleRequest(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
	if m.msgAPI != nil {
		resp = m.msgAPI.HandleRequest(req)
	}
	if resp == nil {
		resp = m.HiveModuleBase.HandleRequest(req)
	}
	return resp
}

// Start readies the module for use using the given yaml configuration.
//
// This:
// - opens the bucket store using the configured name.
// - enable the messaging request handler
// - enable the http request handler using the given router
// - updates this service TD in the store
//
// yamlConfig contains the settings to use.
func (m *DirectoryModule) Start() (err error) {
	storageDir := ""
	if m.storageRoot != "" {
		storageDir = filepath.Join(m.storageRoot, m.ModuleID)
		m.bucketStore = kvbtree.NewKVStore(storageDir)
	} else {
		m.bucketStore = kvbtree.NewKVStore("")
	}
	err = m.bucketStore.Open()
	if err == nil {
		bucketName := m.ModuleID
		m.service, err = service.StartDirectoryService(m.bucketStore, bucketName)
	}
	if err == nil {
		m.msgAPI = api.NewDirectoryMsgHandler(m.ModuleID, m.service)
	}
	if err == nil {
		m.restAPI = api.StartDirectoryRestHandler(m.service, m.router)
	}
	return err
}

// Stop any running actions
func (m *DirectoryModule) Stop() {
	m.service.Stop()
	m.bucketStore.Close()
}

// Start a new directory server module.
// On start this opens or creates a directory store in root/moduleID.
// Directory entries are stored in the 'directory' bucket.
// If a router is provided this registers the HTTP API with the router.
//
// storageRoot is the root dir of the storage area. Use "" for testing with an in-memory store.
// router is the html server router to register the html API handlers with.
func NewDirectoryModule(storageRoot string, router *chi.Mux) *DirectoryModule {

	m := &DirectoryModule{
		HiveModuleBase: modules.HiveModuleBase{
			ModuleID:   directory.DefaultDirectoryThingID,
			Properties: make(map[string]any),
		},
		storageRoot: storageRoot,
		router:      router,
	}
	return m
}
