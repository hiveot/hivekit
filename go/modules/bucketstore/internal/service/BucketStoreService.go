// package service with the bucket store module code
package service

import (
	_ "embed"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules"
	bucketstoreapi "github.com/hiveot/hivekit/go/modules/bucketstore/api"
	"github.com/hiveot/hivekit/go/modules/bucketstore/internal/stores"
	"github.com/hiveot/hivekit/go/modules/bucketstore/internal/stores/kvbtree"
)

// storage name and thingID
const DefaultBucketStoreThingID = "bucketstore"

// BucketStoreService is a module for providing a persistent key-value storage
// for remote services and bindings.
// It is primarily intended for shared storage under 1GB used by one or more
// services.
//
// The module is configured using yaml.
type BucketStoreService struct {
	modules.HiveModuleBase

	// Storage type from config, kvbtree for small stores <100MB) or pebble for big ones
	// The default is kvbtree.
	StoreType string `yaml:"storeType"`

	// The storage data file, folder or URL.
	// When empty, a non-persistent in-memory kvbtree store will be used. (mostly for testing)
	location string

	// The storage bucket store itself, kvbtree, pebble or the default, the pipeline store.
	store  bucketstoreapi.IBucketStorage
	bucket bucketstoreapi.IBucket

	// the WoT messaging API
	msgAPI *BucketMsgHandler
}

func (m *BucketStoreService) GetService() bucketstoreapi.IBucketStorage {
	return m.store
}

// HandleRequest passes the module request messages to the API handler.
func (m *BucketStoreService) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
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
// This creates a bucket store in {storageDir} and enables the messaging request handler.
//
// yamlConfig with optional configuration (todo)
func (m *BucketStoreService) Start(yamlConfig string) (err error) {

	// if a storage directory is provided then open a store
	// otherwise create an in-memory store.
	if m.location != "" {
		m.store, err = stores.NewBucketStorage(m.location, m.StoreType)
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
func (m *BucketStoreService) Stop() {
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
// If an embedded store is used then the history data is stored in the storageDir directory,
// or "" for testing with in-memory storage.
//
// location is the bucket storage file, directory or URL depending on the type
func NewBucketStoreService(location string, storeType string) *BucketStoreService {

	m := &BucketStoreService{
		HiveModuleBase: modules.HiveModuleBase{},
		location:       location,
		StoreType:      storeType,
		// StoreName:   defaultStoreName,
		// bucketStore: bucketStore,
	}
	m.SetModuleID(DefaultBucketStoreThingID)

	var _ modules.IHiveModule = m                 // interface check
	var _ bucketstoreapi.IBucketStorage = m.store // interface check

	return m
}
