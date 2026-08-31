package kvbtreestore

import (
	"github.com/hiveot/hivekit/go/cells/bucketstore"
	"github.com/hiveot/hivekit/go/cells/bucketstore/kvbtreestore/internal"
)

// OpenKVBTreeStore is the factory function that creates a new, unopened, instance of a bucketstore
// using the kvbtree library.
// Use Close() to release its resources.
//
// Intended for use by the bucketstore service and for cells that need embedded storage.
//
// Note: intended for supporting configurable backends. If the backend type is fixed
// then it is better to create an instance of that backend directly to reduce compile size.
//
// location is the data directory or URL where the data persists. Use "" for an in-memory btree
func OpenKVBTreeStore(location string) (store bucketstore.IBucketStore, err error) {

	return internal.OpenKVBtreeStore(location)
}
