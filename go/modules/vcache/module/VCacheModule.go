package vcachemodule

import (
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/vcache"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
)

// VCacheModule is the value-cache module implementation
// this implements the IVCache and IHiveModule interface
type VCacheModule struct {
	modules.HiveModuleBase

	// The maximum age a cached value is valid for. Anything old results in requesting
	// the value from the device. The default is 24 hours.
	maxAgeSec int

	// map of thingID/name/affordance to cached value
	store VCacheStore
}

func (m *VCacheModule) GetCacheStatus() vcache.CacheInfo {
	info := vcache.CacheInfo{
		NrThings:  m.store.GetNrThings(),
		MaxAgeSec: m.maxAgeSec,
	}
	return info
}

// HandleNotification passes notifications upstream after storing the values for query requests
func (m *VCacheModule) HandleNotification(notif *msg.NotificationMessage) {

	switch notif.AffordanceType {
	case msg.AffordanceTypeProperty:
		m.store.WriteProperty(notif.ThingID, notif.Name, notif.Data)
	}

	m.ForwardNotification(notif)
}

// HandleRequest responds with request queries for Things whose values have been cached.
//
// If the value is not cached the request is forwarded down the chain.
// Currently, only notifications can populate the cache to ensure it remains up to date.
func (m *VCacheModule) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var isCached bool
	var value any

	// handle read requests
	switch req.Operation {
	case wot.OpReadProperty:
		value, isCached = m.ReadProperty(req.ThingID, req.Name)
	case wot.OpReadMultipleProperties:
		var names []string
		err := utils.DecodeAsObject(req.Input, &names)
		if err != nil {
			// missing names
			slog.Warn("HandleRequest: ReadMultipleProperties, missing names. Forwarding the request",
				"senderID", req.SenderID, "thingID", req.ThingID)
		} else {
			value, isCached = m.ReadMultipleProperties(req.ThingID, names)
		}

	case wot.OpReadAllProperties:
		// this is currently not supported as there is no knowledge of all possible properties
		// value, isCached = m.ReadallProperties(req.ThingID)
	}

	if !isCached {
		return m.ForwardRequest(req, replyTo)
	}

	// check age
	// a cached value was found
	resp := req.CreateResponse(value, nil)
	return replyTo(resp)
}

// ReadAllProperties returns all known cached properties of the thing.
// Since the cache doesn't have the TD it can only assume that subscription is made on all properties
func (m *VCacheModule) ReadAllProperties(thingID string) (v any, isCached bool) {
	return nil, false
}

// ReadProperty returns the cached value of a property.
// If the property value has expired then it is removed from the cache.
func (m *VCacheModule) ReadProperty(thingID string, name string) (v any, isCached bool) {

	cv, found := m.store.ReadProperty(thingID, name)
	if !found {
		return nil, false
	}
	// if the value is expired, remove it
	maxAge := time.Second * time.Duration(m.maxAgeSec)
	if cv.Timestamp.Add(maxAge).Before(time.Now()) {
		m.store.RemoveProperty(thingID, name)
		return nil, false
	}
	return cv.Value, true
}

// ReadMultipleProperties returns the cached value of cached properties.
func (m *VCacheModule) ReadMultipleProperties(thingID string, names []string) (v any, isCached bool) {
	return nil, false
}

// SetCacheValidity sets the duration a cached value is valid for.
// Cached values older than this will be removed instead of returned.
func (m *VCacheModule) SetCacheValidity(maxAgeSec int) {
	m.maxAgeSec = maxAgeSec
}

// Start opens the logging destination.
func (m *VCacheModule) Start(configYaml string) (err error) {
	return err
}

// Stop closes the logging destination.
func (m *VCacheModule) Stop() {
}

// Create a new instance of the value cache module.
func NewVCacheModule() *VCacheModule {

	m := &VCacheModule{
		maxAgeSec: vcache.DefaultVCacheMaxAgeSec,
		store:     *NewVCacheStore(),
	}

	var _ vcache.IVCacheModule = m // interface check
	return m
}
