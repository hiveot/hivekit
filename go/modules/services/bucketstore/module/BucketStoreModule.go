// package module with the directory module factory
package module

import (
	_ "embed"
	"fmt"
	"path/filepath"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/services/bucketstore"
	"github.com/hiveot/hivekit/go/modules/services/bucketstore/api"
	"github.com/hiveot/hivekit/go/modules/services/bucketstore/kvbtree"
	"github.com/hiveot/hivekit/go/modules/services/bucketstore/pebble"
	"github.com/hiveot/hivekit/go/msg"
)

// storage name and thingID
const DefaultBucketStoreThingID = "bucketstore"

// BucketStoreModule is a module for providing a persistent key-value storage
// for remote services and bindings.
// It is primarily intended for shared storage under 1GB used by one or more
// services.
//
// The module is configured using yaml.
type BucketStoreModule struct {
	modules.HiveModuleBase

	// Storage type from config, kvbtree for small stores <100MB) or pebble for big ones
	// The default is kvbtree.
	StoreType string `yaml:"storeType"`

	// The persistence data directory root.
	// When empty, a non-persistent in-memory kvbtree store will be used. (mostly for testing)
	storageRoot string

	// The storage bucket store itself, kvbtree, pebble or the default, the pipeline store.
	store  bucketstore.IBucketStore
	bucket bucketstore.IBucket

	// temporary cursors for remote iterators
	cursorCache *bucketstore.CursorCache

	// the WoT messaging API
	msgAPI *api.BucketMsgHandler
}

func (m *BucketStoreModule) GetService() bucketstore.IBucketStore {
	return m.store
}

// HandleRequest passes the module request messages to the API handler.
func (m *BucketStoreModule) HandleRequest(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
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
// This creates a bucket store in {storeRoot}/{moduleID} and enables the
// messaging request handler.
func (m *BucketStoreModule) Start() (err error) {

	// if a storage directory is provided then open a store under the given name.
	// otherwise create an in-memory store.
	if m.storageRoot != "" {
		storeDirectory := filepath.Join(m.storageRoot, m.ModuleID)
		switch m.StoreType {
		case bucketstore.BackendKVBTree:
			m.store = kvbtree.NewKVStore(storeDirectory)
		case bucketstore.BackendPebble:
			m.store = pebble.NewPebbleStore(storeDirectory)
		default:
			// unknown storage type, use in-memory
			err = fmt.Errorf("unknown storage type '%s'", m.StoreType)
			return err
		}
	} else {
		// no persistence. Use an in-memory store
		m.store = kvbtree.NewKVStore("")
	}
	err = m.store.Open()
	if err == nil {
		m.msgAPI = api.NewBucketMsgHandler(m.ModuleID, m.store)
	}
	// for remote iterators
	m.cursorCache = bucketstore.NewCursorCache()

	return err
}

// Stop any running actions
func (m *BucketStoreModule) Stop() {
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
// storageRoot is the application storage root directory, "" for testing with in-memory storage
func NewBucketStoreModule(storageRoot string) *BucketStoreModule {

	m := &BucketStoreModule{
		HiveModuleBase: modules.HiveModuleBase{
			ModuleID:   DefaultBucketStoreThingID,
			Properties: make(map[string]any),
		},
		storageRoot: storageRoot,
		// StoreType:   defaultStoreType,
		// StoreName:   defaultStoreName,
		// bucketStore: bucketStore,
	}
	var _ modules.IHiveModule = m // interface check

	return m
}
