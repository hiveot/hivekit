// package service with the bucket store module code
package internal

import (
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules"
	bucketstore "github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/modules/bucketstore/kvbtreestore"
	"github.com/hiveot/hivekit/go/modules/bucketstore/pebblestore"
)

// BucketServiceImpl is a module for providing a persistent key-value storage
// for remote services and bindings.
// It is primarily intended for shared storage under 1GB used by one or more
// services.
//
// The module is configured using yaml.
type BucketServiceImpl struct {
	*modules.HiveModuleBase

	// Storage type from config, kvbtree for small stores <100MB) or pebble for big ones
	// The default is kvbtree.
	backend string `yaml:"storeType"`

	// The storage data file, folder or URL.
	// When empty, a non-persistent in-memory kvbtree store will be used. (mostly for testing)
	location string

	// The storage bucket store itself, kvbtree, pebble or the default, the pipeline store.
	store  bucketstore.IBucketStore
	bucket bucketstore.IBucket
	// serving cursor requests
	cursorCache *CursorCache
}

func (svc *BucketServiceImpl) GetService() bucketstore.IBucketStore {
	return svc.store
}

// HandleRequest passes the module request messages to the API handler.
func (svc *BucketServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
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
func (svc *BucketServiceImpl) Start() (err error) {

	slog.Info("Start: Starting bucketstore module")

	// if a storage directory is provided then open a store
	// otherwise create an in-memory store.
	backend := svc.backend

	// if no location so use the in-memory store
	if svc.location == "" {
		backend = bucketstore.BackendKVBTree
	}
	switch backend {
	case bucketstore.BackendKVBTree:
		svc.store = kvbtreestore.NewBucketStore(svc.location)
	case bucketstore.BackendPebble:
		svc.store = pebblestore.NewBucketStore(svc.location)
	default:
		// unknown storage type
		err = fmt.Errorf("unknown storage type '%s'", backend)
		return err
	}
	err = svc.store.Open()
	return err
}

// Stop any running actions
func (svc *BucketServiceImpl) Stop() {
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
func NewBucketServiceImpl(location string, storeType string) *BucketServiceImpl {

	// this module is a singleton that exposes multiple service things
	thingID := bucketstore.DefaultBucketStoreThingID
	m := &BucketServiceImpl{
		HiveModuleBase: modules.NewHiveModuleBase(thingID, 0),
		location:       location,
		backend:        storeType,
		cursorCache:    NewCursorCache(),

		// StoreName:   defaultStoreName,
		// bucketStore: bucketStore,
	}

	var _ api.IHiveModule = m                // interface check
	var _ bucketstore.IBucketStore = m.store // interface check

	return m
}
