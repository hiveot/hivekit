package stores

import (
	"fmt"

	"github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/modules/bucketstore/stores/kvbtree"
	"github.com/hiveot/hivekit/go/modules/bucketstore/stores/pebble"
)

// NewBucketStore creates a new, unopened, instance of a bucketstore for the given location
// and type. Open must be called before use. (or use OpenBucketStore)
//
// Intended for the bucketstore module and for modules that need internal storage.
//
// Note: intended for supporting configurable backends. If the backend type is fixed
// then it is better to create an instance of that backend directly to reduce compile size.
//
// storeDirectory is the data directory to store the code. Only for embedded backends.
// backend, one of the supported backends (BackendKVBTree, BackendPebble)
func NewBucketStore(storeDirectory string, backend string) (
	store bucketstore.IBucketStore, err error) {

	switch backend {
	case bucketstore.BackendKVBTree:
		store = kvbtree.NewKVStore(storeDirectory)
	case bucketstore.BackendPebble:
		store = pebble.NewPebbleStore(storeDirectory)
	default:
		// unknown storage type
		err = fmt.Errorf("unknown storage type '%s'", backend)
		return nil, err
	}
	return store, err
}

// Convenience function that creates and opens a store
func OpenBucketStore(storeDirectory string, backend string) (
	store bucketstore.IBucketStore, err error) {

	store, err = NewBucketStore(storeDirectory, backend)
	if err == nil {
		err = store.Open()
	}
	return store, err
}
