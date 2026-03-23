package bucketstore

import (
	bucketstoreapi "github.com/hiveot/hivekit/go/modules/bucketstore/api"
	"github.com/hiveot/hivekit/go/modules/bucketstore/internal/service"
)

// NewBucketStoreService returns a new bucket store service module
func NewBucketStoreService(
	location string, storeType string) bucketstoreapi.IBucketStoreService {

	m := service.NewBucketStoreService(location, storeType)
	return m
}

// CursorCache manages a set of cursors that can be addressed remotely by key.
// Intended for servers that let remote clients iterate a cursor in the bucket store.
func NewCursorCache() bucketstoreapi.ICursorCache {
	return service.NewCursorCache()
}
