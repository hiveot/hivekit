package VCacheModule

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	vcacheapi "github.com/hiveot/hivekit/go/modules/vcache/api"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
)

// VCacheModule is the notification-cache module implementation
// this implements the INCache and IHiveModule interface
type VCacheModule struct {
	modules.HiveModuleBase

	// map of thingID/name/affordance to cached value
	store VCacheStore
}

func (m *VCacheModule) GetCacheStatus() vcacheapi.CacheInfo {
	info := vcacheapi.CacheInfo{
		NrThings: m.store.GetNrThings(),
	}
	return info
}

// HandleNotification passes notifications upstream after storing the values for query requests
func (m *VCacheModule) HandleNotification(notif *msg.NotificationMessage) {

	switch notif.AffordanceType {
	case msg.AffordanceTypeEvent:
		m.store.WriteEvent(notif)
	case msg.AffordanceTypeProperty:
		m.store.WriteProperty(notif)
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
	// wot doesnt define operations for reading events
	case wot.HTOpReadEvent:
		notif := m.ReadEvent(req.ThingID, req.Name)
		if notif != nil {
			isCached = true
			value = notif.Data
		}
	case wot.OpReadProperty:
		notif := m.ReadProperty(req.ThingID, req.Name)
		if notif != nil {
			isCached = true
			value = notif.Data
		}
	case wot.OpReadMultipleProperties:
		var names []string
		err := utils.DecodeAsObject(req.Input, &names)
		if err != nil {
			// missing names
			slog.Warn("HandleRequest: ReadMultipleProperties, missing names. Forwarding the request",
				"senderID", req.SenderID, "thingID", req.ThingID)
		} else {
			// this only succeeds if all requested properties are available
			notifMap, hasAllNames := m.ReadMultipleProperties(req.ThingID, names)
			if hasAllNames {
				propMap := make(map[string]any)
				isCached = true
				for name, notif := range notifMap {
					propMap[name] = notif.Data
				}
				value = propMap
			}
		}

	case wot.OpReadAllProperties:
		// querying all properties is not supported as there is no knowledge of all possible
		// properties.
		isCached = false
	}

	if !isCached {
		// forward the request to the actual device
		return m.ForwardRequest(req, replyTo)
	}

	// a cached notification was found
	resp := req.CreateResponse(value, nil)
	return replyTo(resp)
}

// ReadAllProperties returns all known cached properties of the thing.
// Since the cache doesn't have the TD it can only assume that subscription is made on all properties
// func (m *VCacheModule) ReadAllProperties(thingID string) (v any, isCached bool) {
// 	return nil, false
// }

// ReadEvent returns the latest cached event value.
func (m *VCacheModule) ReadEvent(thingID string, name string) (notif *msg.NotificationMessage) {

	notif = m.store.ReadEvent(thingID, name)
	return notif
}

// ReadProperty returns the last known cached notification of a property.
func (m *VCacheModule) ReadProperty(thingID string, name string) (notif *msg.NotificationMessage) {

	notif = m.store.ReadProperty(thingID, name)
	return notif
}

// ReadMultipleProperties returns the value of cached properties.
// This returns a map of available values and a 'isCached' flag if all values are available.
// If not all requested values are available then isCached is false.
func (m *VCacheModule) ReadMultipleProperties(
	thingID string, names []string) (v map[string]*msg.NotificationMessage, isCached bool) {

	propMap, isCached := m.store.ReadMultipleProperties(thingID, names)
	return propMap, isCached
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
		store: *NewNCacheStore(),
	}
	m.SetModuleID(vcacheapi.DefaultVCacheModuleID)
	var _ vcacheapi.IVCacheModule = m // interface check
	return m
}
