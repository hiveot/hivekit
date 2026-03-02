package bucketstore

import (
	"fmt"

	bucketstoreapi "github.com/hiveot/hivekit/go/modules/bucketstore/api"
	"github.com/hiveot/hivekit/go/modules/bucketstore/internal/stores/kvbtree"
	"github.com/hiveot/hivekit/go/modules/bucketstore/internal/stores/pebble"
)

// NewBucketStore is the factory function that creates a new, unopened, instance of a bucketstore
// for the given location and type. Open must be called before use. (or use OpenBucketStore)
//
// Intended for the bucketstore module and for modules that need internal storage.
//
// Note: intended for supporting configurable backends. If the backend type is fixed
// then it is better to create an instance of that backend directly to reduce compile size.
//
// location is the data directory or URL where the data persists. Only for embedded backends.
// backend, one of the supported backends (BackendKVBTree, BackendPebble)
func NewBucketStore(location string, backend string) (
	store bucketstoreapi.IBucketStore, err error) {

	switch backend {
	case bucketstoreapi.BackendKVBTree:
		store = kvbtree.NewKVStore(location)
	case bucketstoreapi.BackendPebble:
		store = pebble.NewPebbleStore(location)
	default:
		// unknown storage type
		err = fmt.Errorf("unknown storage type '%s'", backend)
		return nil, err
	}
	return store, err
}

// Convenience function that creates and opens a store
func OpenBucketStore(storeDirectory string, backend string) (
	store bucketstoreapi.IBucketStore, err error) {

	store, err = NewBucketStore(storeDirectory, backend)
	if err == nil {
		err = store.Open()
	}
	return store, err
}
