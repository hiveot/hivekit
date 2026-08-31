package bucketstore_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/bucketstore"
	"github.com/hiveot/hivekit/go/cells/bucketstore/internal"
)

// StartBucketStoreService returns a new bucket store service
// Intended to be used as a local or remote accessible storage facility.
// See also StartCursorCache() to manage cursor lifecycle for remote use.
//
//	location is the storage directory
//	storeType is the backend type, eg BackendInMemory, BackendKVBTree, BackendPebble,...
func StartBucketStoreService(
	location string, storeType string) (bucketstore.IBucketStoreService, error) {

	// if location == "" {
	// 	backend = bucketstore.BackendKVBTree
	// }
	// switch backend {
	// case bucketstore.BackendKVBTree:
	// 	store = kvbtree.NewKVStore(location)
	// case bucketstore.BackendPebble:
	// 	store = pebble.NewPebbleStore(location)
	// default:
	// 	// unknown storage type
	// 	err = fmt.Errorf("unknown storage type '%s'", backend)
	// 	return nil, err
	// }

	svc, err := internal.StartBucketServiceImpl(location, storeType)
	return svc, err
}

// StartBucketStoreServiceFactory start a new bucket store service using the factory environment
// This defaults to the kvbtree store which is a balance between speed and capacity.
func StartBucketStoreServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	location := f.GetEnvironment().GetStorageDir(bucketstore.BucketStoreCellType)
	// TODO: support configuration of storage type (default is pebble)
	svc, err := StartBucketStoreService(location, bucketstore.BackendKVBTree)
	return svc, err
}

// StartCursorCache manages a set of cursors that can be addressed remotely by key.
// Intended for servers that let remote clients iterate a cursor in the bucket store.
// Call Stop to end the background process and free resources.
func StartCursorCache() bucketstore.ICursorCache {
	return internal.StartCursorCache()
}
