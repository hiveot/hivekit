package module

import (
	"github.com/hiveot/hivekit/go/modules"
	vcacheapi "github.com/hiveot/hivekit/go/modules/vcache/api"
	"github.com/hiveot/hivekit/go/msg"
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
