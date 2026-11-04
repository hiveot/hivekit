package vcache

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
)

const ValueCacheCellType = "valuecache"
const ValueCacheDefaultThingID = "valuecache"

// The default thingID of this vcache for messaging (if any)
// const DefaultVCacheThingID = "vcache"  - does not handle requests for itself

type CacheInfo struct {

	// The number of things that have cached values
	NrThings int `json:"nrThings"`
}

// IValueCacheService value-cache service interface.
// This service caches notification of properties and event and returns the cached value when a
// ReadProperty operation is received. Intended for speeding up querying property values
// and to provide answers when the queried thing is unavailable.
type IValueCacheService interface {
	api.IHiveCell

	// Forward the request downstream if the service can't answer it.
	// The intent is to pass it to the thing device to serve the request.
	// This typically passes it to a router of some sort.
	ForwardRequest(request *msg.RequestMessage, replyTo msg.ResponseHandler) error

	// Return the current cache status
	GetCacheStatus() CacheInfo

	// Retrieve cached Thing action status or nil if none found
	// Action status are stored in a notification
	ReadAction(thingID string, name string) (actionStatus msg.ActionStatus, found bool)

	// Retrieve cached Thing event notification value or nil if none found
	ReadEvent(thingID string, name string) (tv msg.ThingValue, found bool)

	// Retrieve cached Thing property notification value
	// This returns the cached property notification or nil if not cached.
	ReadProperty(thingID string, name string) (tv msg.ThingValue, found bool)

	// ReadMultipleProperties returns the value of cached properties.
	// This returns a map of available values and a 'foundAll' flag if all values are available.
	// If some but not all requested values are available then foundAll is false.
	ReadMultipleProperties(
		thingID string, names []string) (v map[string]msg.ThingValue, foundAll bool)

	// WriteValue updates the latest property, event or action status in the vcache
	WriteValue(notif *msg.NotificationMessage)
}
