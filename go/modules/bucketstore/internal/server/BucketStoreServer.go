// package module with the directory module factory
package bucketstoreserver

import (
	_ "embed"
	"path/filepath"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketstoreapi "github.com/hiveot/hivekit/go/modules/bucketstore/api"
	"github.com/hiveot/hivekit/go/modules/bucketstore/internal/stores/kvbtree"
	"github.com/hiveot/hivekit/go/msg"
)

// storage name and thingID
const DefaultBucketStoreThingID = "bucketstore"

// BucketStoreServer is a module for providing a persistent key-value storage
// for remote services and bindings.
// It is primarily intended for shared storage under 1GB used by one or more
// services.
//
// The module is configured using yaml.
type BucketStoreServer struct {
	modules.HiveModuleBase

	// Storage type from config, kvbtree for small stores <100MB) or pebble for big ones
	// The default is kvbtree.
	StoreType string `yaml:"storeType"`

	// The persistence data directory root.
	// When empty, a non-persistent in-memory kvbtree store will be used. (mostly for testing)
	storageRoot string

	// The storage bucket store itself, kvbtree, pebble or the default, the pipeline store.
	store  bucketstoreapi.IBucketStore
	bucket bucketstoreapi.IBucket

	// the WoT messaging API
	msgAPI *BucketMsgHandler
}

func (m *BucketStoreServer) GetService() bucketstoreapi.IBucketStore {
	return m.store
}

// HandleRequest passes the module request messages to the API handler.
func (m *BucketStoreServer) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var resp *msg.ResponseMessage
	if m.msgAPI != nil {
		resp = m.msgAPI.HandleRequest(req)
	}
	if resp == nil {
		err = m.HiveModuleBase.HandleRequest(req, replyTo)
	} else {
		err = replyTo(resp)
	}
	return err
}

// Start readies the module for use using the given yaml configuration.
//
// This creates a bucket store in {storeRoot}/{moduleID} and enables the
// messaging request handler.
//
// yamlConfig with optional configuration (todo)
func (m *BucketStoreServer) Start(yamlConfig string) (err error) {

	// if a storage directory is provided then open a store under the given name.
	// otherwise create an in-memory store.
	if m.storageRoot != "" {
		storeDirectory := filepath.Join(m.storageRoot, m.GetModuleID())
		m.store, err = bucketstore.NewBucketStore(storeDirectory, m.StoreType)
	} else {
		// no persistence. Use an in-memory store
		m.store = kvbtree.NewKVStore("")
	}
	err = m.store.Open()
	if err == nil {
		m.msgAPI = NewBucketMsgHandler(m.GetModuleID(), m.store)
	}
	// for remote iterators
	// m.cursorCache = NewCursorCache()

	return err
}

// Stop any running actions
func (m *BucketStoreServer) Stop() {
	if m.bucket != nil {
		m.bucket.Close()
		m.bucket = nil
	}
	if m.store != nil {
		m.store.Close()
		m.store = nil
	}
}

// Start a new bucket storage instance
// Run Start() before use.
//
// If an embedded store is used then the history data is stored in the directory
// {storageRoot}/{moduleID}.
//
// storageRoot is the application storage root directory, "" for testing with in-memory storage
func NewBucketStoreServer(storageRoot string, storeType string) *BucketStoreServer {

	m := &BucketStoreServer{
		HiveModuleBase: modules.HiveModuleBase{},
		storageRoot:    storageRoot,
		StoreType:      storeType,
		// StoreName:   defaultStoreName,
		// bucketStore: bucketStore,
	}
	m.SetModuleID(DefaultBucketStoreThingID)

	var _ modules.IHiveModule = m               // interface check
	var _ bucketstoreapi.IBucketStore = m.store // interface check

	return m
}
