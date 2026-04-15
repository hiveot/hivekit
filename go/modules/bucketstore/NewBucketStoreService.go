package bucketstore

import (
	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules"
	bucketstoreapi "github.com/hiveot/hivekit/go/modules/bucketstore/api"
	"github.com/hiveot/hivekit/go/modules/bucketstore/internal/service"
)

// NewBucketStoreService returns a new bucket store service module
// Intended to be used as a remote accessible storage facility.
//
//	location is the storage directory
//	storeType is the backend type, eg BackendInMemory, BackendKVBTree, BackendPebble,...
func NewBucketStoreService(
	location string, storeType string) bucketstoreapi.IBucketStoreService {

	m := service.NewBucketStoreService(location, storeType)
	return m
}

// NewBucketStoreServiceFactory returns a new bucket store service using the factory environment
// This defaults to the kvbtree store which is a balance between speed and capacity.
func NewBucketStoreServiceFactory(f factoryapi.IModuleFactory) modules.IHiveModule {

	location := f.GetEnvironment().GetStorageDir(bucketstoreapi.BucketStoreModuleType)
	// TODO: support configuration of storage type (default is pebble)
	m := NewBucketStoreService(location, bucketstoreapi.BackendKVBTree)
	return m
}

// CursorCache manages a set of cursors that can be addressed remotely by key.
// Intended for servers that let remote clients iterate a cursor in the bucket store.
func NewCursorCache() bucketstoreapi.ICursorCache {
	return service.NewCursorCache()
}
