package module

import (
	"github.com/hiveot/hivekit/go/modules"
	vcacheapi "github.com/hiveot/hivekit/go/modules/vcache/api"
	"github.com/hiveot/hivekit/go/msg"
)

// VCacheServer is the value-cache server module implementation
// this implements the IVCache and IHiveModule interface
type VCacheServer struct {
	modules.HiveModuleBase

	msgAPI *VCacheMsgHandler

	// map of thingID/name/affordance to cached value
	store VCacheStore
}

func (m *VCacheServer) GetCacheStatus() vcacheapi.CacheInfo {
	info := vcacheapi.CacheInfo{
		NrThings: m.store.GetNrThings(),
	}
	return info
}

// HandleNotification passes notifications upstream after storing the values for query requests
func (m *VCacheServer) HandleNotification(notif *msg.NotificationMessage) {
	// cache the notification values
	m.msgAPI.HandleNotification(notif)

	// forward the notification up the chain
	m.ForwardNotification(notif)
}

// HandleRequest handles requests for reading digital twin values
func (m *VCacheServer) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	return m.msgAPI.HandleRequest(req, replyTo)
}

// ReadAction returns the latest cached action status.
func (m *VCacheServer) ReadAction(thingID string, name string) (notif *msg.NotificationMessage) {

	notif = m.store.ReadAction(thingID, name)
	return notif
}

// ReadEvent returns the latest cached event value.
func (m *VCacheServer) ReadEvent(thingID string, name string) (notif *msg.NotificationMessage) {

	notif = m.store.ReadEvent(thingID, name)
	return notif
}

// ReadProperty returns the last known cached notification of a property.
func (m *VCacheServer) ReadProperty(thingID string, name string) (notif *msg.NotificationMessage) {

	notif = m.store.ReadProperty(thingID, name)
	return notif
}

// ReadMultipleProperties returns the value of cached properties.
// This returns a map of available values and a 'isCached' flag if all values are available.
// If not all requested values are available then isCached is false.
func (m *VCacheServer) ReadMultipleProperties(
	thingID string, names []string) (v map[string]*msg.NotificationMessage, isCached bool) {

	propMap, isCached := m.store.ReadMultipleProperties(thingID, names)
	return propMap, isCached
}

// Start opens the logging destination.
func (m *VCacheServer) Start(configYaml string) (err error) {
	m.msgAPI = NewVCacheMsgHandler(m)
	return err
}

// Stop closes the logging destination.
func (m *VCacheServer) Stop() {
}

// WriteEvent updates the event in the vcache
func (m *VCacheServer) WriteAction(req *msg.RequestMessage) {
	notif := msg.NewNotificationMessage(
		req.SenderID, msg.AffordanceTypeAction, req.ThingID, req.Name, req.Input)
	notif.Timestamp = req.Created
	notif.CorrelationID = req.CorrelationID
	m.store.WriteValue(notif)
}

// WriteEvent updates the event in the vcache
func (m *VCacheServer) WriteEvent(notif *msg.NotificationMessage) {
	m.store.WriteValue(notif)
}

// WriteProperty updates the property in the vcache
func (m *VCacheServer) WriteProperty(notif *msg.NotificationMessage) {
	m.store.WriteValue(notif)
}

// Create a new instance of the value cache module.
func NewVCacheServer() *VCacheServer {

	m := &VCacheServer{
		store: *NewNCacheStore(),
	}
	m.SetModuleID(vcacheapi.DefaultVCacheModuleID)
	var _ vcacheapi.IVCacheServer = m // interface check
	return m
}
