package pebblestore

import (
	"github.com/hiveot/hivekit/go/cells/bucketstore"
	"github.com/hiveot/hivekit/go/cells/bucketstore/pebblestore/internal"
)

// OpenPebbleStore is the factory function that creates a new, opened, instance of a bucketstore
// using the pebble library.
//
// Intended for use by the bucketstore service and for cells that need embedded storage.
//
// Note: intended for supporting configurable backends. If the backend type is fixed
// then it is better to create an instance of that backend directly to reduce compile size.
//
// location is the data directory or URL where the data persists. Use "" for an in-memory btree
func OpenPebbleStore(location string) (store bucketstore.IBucketStore, err error) {

	store, err = internal.OpenPebbleStore(location)
	return store, err
}
