package vcacheapi

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/msg"
)

// The default module instanceID
// When using multiple instances, it is recommended to set a different ID using
// SetModuleID()
const DefaultVCacheModuleID = "vcache"

type CacheInfo struct {

	// The number of things that have cached values
	NrThings int `json:"nrThings"`
}

// IVCache value-cache module interface.
// This module caches notification of properties and event and returns the cached value when a
// ReadProperty operation is received. Intended for speeding up querying property values
// and to provide answers when the queried thing is unavailable.
type IVCacheModule interface {
	modules.IHiveModule

	// Return the current cache status
	GetCacheStatus() CacheInfo

	// Retrieve cached Thing event notification value or nil if none found
	ReadEvent(thingID string, name string) (eventNotif *msg.NotificationMessage)

	// Retrieve cached Thing property notification value
	// This returns the cached property notification or nil if not cached.
	ReadProperty(thingID string, name string) (propNotif *msg.NotificationMessage)

	// 	// Retrieve cached Thing property values
	// GetProperties(thingID string) map[string]any
}
