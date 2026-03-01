package vcache

import (
	"github.com/hiveot/hivekit/go/modules"
)

// The default module instanceID
// When using multiple instances, it is recommended to set a different ID using
// SetModuleID()
const DefaultVCacheModuleID = "vcache"

// The default maximum age of cached values after which they are removed
const DefaultVCacheMaxAgeSec = 24 * 3600

type CacheInfo struct {

	// The number of things that have cached values
	NrThings int `json:"nrThings"`

	// The maximum age of cached values in seconds
	MaxAgeSec int `json:"maxAgeSec"`
}

// IVCache value-cache module interface.
// This module caches notification values and returns the cached value when a ReadProperty operation
// is received. Intended for speeding up querying property values and to provide answers when the
// queried thing is unavailable.
type IVCacheModule interface {
	modules.IHiveModule

	// Return the current cache status
	GetCacheStatus() CacheInfo

	// Retrieve cached Thing event values
	// GetEvents(thingID string) map[string]any

	// Retrieve cached Thing property value
	// This returns the cached value with isCached is true,
	// or nil with isCached is false if no valid value is found.
	ReadProperty(thingID string, name string) (v any, isCached bool)

	// 	// Retrieve cached Thing property values
	// GetProperties(thingID string) map[string]any

	// SetCacheValidity sets the duration a cached value is valid for in seconds.
	// Cached values older than this will be removed instead of returned.
	SetCacheValidity(maxAgeSec int)
}
