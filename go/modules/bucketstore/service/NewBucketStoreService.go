package bucketstore_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/modules/bucketstore/internal"
)

// NewBucketStoreService returns a new bucket store service module
// Intended to be used as a remote accessible storage facility.
//
//	location is the storage directory
//	storeType is the backend type, eg BackendInMemory, BackendKVBTree, BackendPebble,...
func NewBucketStoreService(
	location string, storeType string) bucketstore.IBucketStoreService {

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

	m := internal.NewBucketServiceImpl(location, storeType)
	return m
}

// NewBucketStoreServiceFactory returns a new bucket store service using the factory environment
// This defaults to the kvbtree store which is a balance between speed and capacity.
func NewBucketStoreServiceFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {

	location := f.GetEnvironment().GetStorageDir(bucketstore.BucketStoreModuleType)
	// TODO: support configuration of storage type (default is pebble)
	m := NewBucketStoreService(location, bucketstore.BackendKVBTree)
	return m, nil
}

// CursorCache manages a set of cursors that can be addressed remotely by key.
// Intended for servers that let remote clients iterate a cursor in the bucket store.
func NewCursorCache() bucketstore.ICursorCache {
	return internal.NewCursorCache()
}
