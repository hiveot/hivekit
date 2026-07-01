// package service with the bucket store module code
package service

import (
	_ "embed"
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules"
	bucketstore "github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/modules/bucketstore/internal/stores"
	"github.com/hiveot/hivekit/go/modules/bucketstore/internal/stores/kvbtree"
)

// BucketStoreService is a module for providing a persistent key-value storage
// for remote services and bindings.
// It is primarily intended for shared storage under 1GB used by one or more
// services.
//
// The module is configured using yaml.
type BucketStoreService struct {
	*modules.HiveModuleBase

	// Storage type from config, kvbtree for small stores <100MB) or pebble for big ones
	// The default is kvbtree.
	StoreType string `yaml:"storeType"`

	// The storage data file, folder or URL.
	// When empty, a non-persistent in-memory kvbtree store will be used. (mostly for testing)
	location string

	// The storage bucket store itself, kvbtree, pebble or the default, the pipeline store.
	store  bucketstore.IBucketStore
	bucket bucketstore.IBucket
	// serving cursor requests
	cursorCache *CursorCache
}

func (svc *BucketStoreService) GetService() bucketstore.IBucketStore {
	return svc.store
}

// HandleRequest passes the module request messages to the API handler.
func (svc *BucketStoreService) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if req.ThingID == svc.GetThingID() {
		err = svc.handleBucketStoreRequest(req, replyTo)
	} else {
		err = svc.HiveModuleBase.HandleRequest(req, replyTo)
	}
	return err
}

// Start readies the module for use using the given yaml configuration.
//
// This creates a bucket store in {storageDir} and enables the messaging request handler.
func (svc *BucketStoreService) Start() (err error) {

	slog.Info("Start: Starting bucketstore module")
	// if a storage directory is provided then open a store
	// otherwise create an in-memory store.
	if svc.location != "" {
		svc.store, err = stores.NewBucketStore(svc.location, svc.StoreType)
	} else {
		// no persistence. Use an in-memory store
		svc.store = kvbtree.NewKVStore("")
	}
	err = svc.store.Open()
	return err
}

// Stop any running actions
func (svc *BucketStoreService) Stop() {
	slog.Info("Stop: Stopping bucketstore module")
	if svc.bucket != nil {
		svc.bucket.Close()
		svc.bucket = nil
	}
	if svc.store != nil {
		svc.store.Close()
		svc.store = nil
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

	// this module is a singleton that exposes multiple service things
	thingID := bucketstore.DefaultBucketStoreThingID
	m := &BucketStoreService{
		HiveModuleBase: modules.NewHiveModuleBase(thingID, 0),
		location:       location,
		StoreType:      storeType,
		cursorCache:    NewCursorCache(),

		// StoreName:   defaultStoreName,
		// bucketStore: bucketStore,
	}

	var _ api.IHiveModule = m                // interface check
	var _ bucketstore.IBucketStore = m.store // interface check

	return m
}
