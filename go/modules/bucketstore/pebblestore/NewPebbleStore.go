package pebblestore

import (
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/modules/bucketstore/pebblestore/internal"
)

// NewBucketStore is the factory function that creates a new, unopened, instance of a bucketstore
// using the pebble library.
//
// Intended for use by the bucketstore service and for modules that need embedded storage.
//
// Note: intended for supporting configurable backends. If the backend type is fixed
// then it is better to create an instance of that backend directly to reduce compile size.
//
// location is the data directory or URL where the data persists. Use "" for an in-memory btree
func NewBucketStore(location string) (store bucketstore.IBucketStore) {

	store = internal.NewPebbleStore(location)
	return store
}
