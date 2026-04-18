package stores

import (
	"fmt"

	bucketstore "github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/modules/bucketstore/internal/stores/kvbtree"
	"github.com/hiveot/hivekit/go/modules/bucketstore/internal/stores/pebble"
)

// NewBucketStorage is the factory function that creates a new, unopened, instance of a bucketstore
// for the given location and type. Open must be called before use. (or use OpenBucketStore)
//
// Intended for use by the bucketstore service and for modules that need internal storage.
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
func NewBucketStorage(location string, backend string) (
	store bucketstore.IBucketStorage, err error) {

	if location == "" {
		backend = bucketstore.BackendKVBTree
	}
	switch backend {
	case bucketstore.BackendKVBTree:
		store = kvbtree.NewKVStore(location)
	case bucketstore.BackendPebble:
		store = pebble.NewPebbleStore(location)
	default:
		// unknown storage type
		err = fmt.Errorf("unknown storage type '%s'", backend)
		return nil, err
	}
	return store, err
}
