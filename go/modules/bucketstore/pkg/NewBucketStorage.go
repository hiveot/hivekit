package bucketstorepkg

import (
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/modules/bucketstore/internal/stores"
)

// NewBucketStore is the factory function that creates a new, unopened, instance of a bucketstore
// for the given location and type. Open must be called before use. (or use OpenBucketStore)
//
// Intended for use by the bucketstore service and for modules that need embedded storage.
//
// Note: intended for supporting configurable backends. If the backend type is fixed
// then it is better to create an instance of that backend directly to reduce compile size.
//
// location is the data directory or URL where the data persists. Use "" for an in-memory btree
//
//	for backend KVBTree this is the full path to the storage file
//	for backend pebble this is the full path to the pebble directory
//	for backend redis this is the redis URL
//	for backend sqlite this is the sqlite DB name
//
// backend, one of the supported backends (BackendKVBTree, BackendPebble)
func NewBucketStore(location string, backend string) (
	store bucketstore.IBucketStorage, err error) {

	return stores.NewBucketStorage(location, backend)
}

// Convenience function that creates and opens a store
// This is the factory function that creates a new, opened, instance of a bucketstore
// for the given location and type. Open must be called before use. (or use OpenBucketStore)
//
// Intended for the bucketstore module and for modules that need embedded storage.
//
// Note: intended for supporting configurable backends. If the backend type is fixed
// then it is better to create an instance of that backend directly to reduce compile size.
//
// location is the data directory or URL where the data persists. Using "" for an in-memory kvbtree.
//
//	for backend KVBTree this is the full path to the storage file
//	for backend pebble this is the full path to the pebble directory
//	for backend redis this is the redis URL
//	for backend sqlite this is the sqlite DB name
//
// backend, one of the supported backends (BackendKVBTree, BackendPebble)
func OpenBucketStore(location string, backend string) (
	store bucketstore.IBucketStorage, err error) {

	store, err = NewBucketStore(location, backend)
	if err == nil {
		err = store.Open()
	}
	return store, err
}
