package bucketstorepkg

import (
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/modules/bucketstore/internal/stores/kvbtree"
)

// NewKVBTreeStore creates a new instance of a KVBTree bucketstore for the given location.
// Open must be called before use. (or use OpenBucketStore)
//
// location is the full path to the storage file "" for an in-memory btree
//
// backend, one of the supported backends (BackendKVBTree, BackendPebble)
func NewKVBTreeStore(location string) (store bucketstore.IBucketStore) {

	return kvbtree.NewKVStore(location)
}
