package internal

import (
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells"
	bucketstore "github.com/hiveot/hivekit/go/cells/bucketstore"
	"github.com/hiveot/hivekit/go/cells/bucketstore/kvbtreestore"
	"github.com/hiveot/hivekit/go/cells/bucketstore/pebblestore"
)

// BucketServiceImpl implements a persistent key-value storage for remote services and bindings.
// It is primarily intended for shared storage under 1GB used by one or more
// services.
//
// The service is configured using yaml.
type BucketServiceImpl struct {
	*cells.HiveCellBase

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

// HandleRequest passes the request messages to the service.
func (svc *BucketServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if req.ThingID == svc.GetThingID() {
		err = svc.handleBucketStoreRequest(req, replyTo)
	} else {
		err = svc.HiveCellBase.HandleRequest(req, replyTo)
	}
	return err
}

// Stop any running actions
func (svc *BucketServiceImpl) Stop() {
	slog.Info("Stop: Stopping bucketstore service")
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
func StartBucketServiceImpl(location string, storeType string) (svc *BucketServiceImpl, err error) {

	slog.Info("Start: Starting bucketstore service")
	var store bucketstore.IBucketStore

	// if a storage directory is provided then open a store
	// if no location for use of the in-memory store
	if location == "" {
		storeType = bucketstore.BackendKVBTree
	}
	switch storeType {
	case bucketstore.BackendKVBTree:
		store, err = kvbtreestore.OpenKVBTreeStore(location)
	case bucketstore.BackendPebble:
		store, err = pebblestore.OpenPebbleStore(location)
	default:
		// unknown storage type
		err = fmt.Errorf("Start: unknown storage type '%s'", storeType)
	}
	if err != nil {
		return nil, err
	}

	// this service is a singleton that exposes multiple service things
	thingID := bucketstore.DefaultBucketStoreThingID
	svc = &BucketServiceImpl{
		HiveCellBase: cells.NewHiveCellBase(thingID, 0),
		location:     location,
		backend:      storeType,
		store:        store,
		cursorCache:  StartCursorCache(),

		// StoreName:   defaultStoreName,
		// bucketStore: bucketStore,
	}

	var _ api.IHiveCell = svc                  // interface check
	var _ bucketstore.IBucketStore = svc.store // interface check

	return svc, err
}
