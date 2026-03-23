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

// IVCacheService value-cache module interface.
// This module caches notification of properties and event and returns the cached value when a
// ReadProperty operation is received. Intended for speeding up querying property values
// and to provide answers when the queried thing is unavailable.
type IVCacheService interface {
	modules.IHiveModule

	// Forward the request downstream if the module can't answer it.
	// The intent is to pass it to the thing device to serve the request.
	// This typically passes it to a router of some sort.
	ForwardRequest(request *msg.RequestMessage, replyTo msg.ResponseHandler) error

	// Return the current cache status
	GetCacheStatus() CacheInfo

	// Retrieve cached Thing action status or nil if none found
	// Action status are stored in a notification
	ReadAction(thingID string, name string) (actionNotif *msg.NotificationMessage)

	// Retrieve cached Thing event notification value or nil if none found
	ReadEvent(thingID string, name string) (eventNotif *msg.NotificationMessage)

	// Retrieve cached Thing property notification value
	// This returns the cached property notification or nil if not cached.
	ReadProperty(thingID string, name string) (propNotif *msg.NotificationMessage)

	// ReadMultipleProperties returns the value of cached properties.
	// This returns a map of available values and a 'isCached' flag if all values are available.
	// If not all requested values are available then isCached is false.
	ReadMultipleProperties(
		thingID string, names []string) (v map[string]*msg.NotificationMessage, isCached bool)

	// WriteAction updates the latest action in the vcache
	WriteAction(req *msg.RequestMessage)

	// WriteEvent updates the latest event in the vcache
	WriteEvent(notif *msg.NotificationMessage)

	// WriteProperty updates the latest property in the vcache
	WriteProperty(notif *msg.NotificationMessage)
}
