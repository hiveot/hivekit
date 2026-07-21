package internal

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/vcache"
	vcacheapi "github.com/hiveot/hivekit/go/modules/vcache"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/teris-io/shortid"
)

// ValueCacheService is the value-cache server module implementation
// this implements the IVCache and IHiveModule interface
//
// This service stores that latest value of property and event notifications.
type ValueCacheService struct {
	*modules.HiveModuleBase

	// map of thingID/name/affordance to cached value
	store ValueStore

	// thingID of this service for handling requests
	// vcacheThingID string
}

func (m *ValueCacheService) GetCacheStatus() vcache.CacheInfo {
	info := vcacheapi.CacheInfo{
		NrThings: m.store.GetNrThings(),
	}
	return info
}

// HandleNotification passes notifications upstream after storing the values for query requests
func (m *ValueCacheService) HandleNotification(notif *msg.NotificationMessage) {

	// cache the notification values

	switch notif.AffordanceType {
	case msg.AffordanceTypeEvent:
		m.WriteEvent(notif)
	case msg.AffordanceTypeProperty:
		m.WriteProperty(notif)
	}
	// forward the notification up the chain
	m.ForwardNotification(notif)
}

// HandleRequest handles requests for use by this module.
// This responds with request queries for Things whose values have been cached.
//
// The cache will populate with received notifications but should only return these when
// a subscription is active to guarantee it remains up to date.
//
// If the value is not cached the request is forwarded down the chain.
// Currently, only notifications can populate the cache to ensure it remains up to date.
//
// Property read operations return the value itself while reading events returns
// the notification itself, as the timestamp is important.
func (m *ValueCacheService) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var isCached bool
	var value any

	// handle read requests
	switch req.Operation {
	// wot doesnt define operations for reading events
	case td.HTOpReadEvent:
		// this is not a wot defined operation
		// return the notification itself because the time of the event is relevant
		notif := m.ReadEvent(req.ThingID, req.Name)
		if notif != nil {
			isCached = true
			value = notif //.Data
		}
	case td.OpReadProperty:
		// WoT specifies that read property returns the latest value
		//
		notif := m.ReadProperty(req.ThingID, req.Name)
		if notif != nil {
			isCached = true
			value = notif.Data
		}
	case td.OpReadMultipleProperties:
		// WoT specifies that ReadMultipleProperties returns a map of [name]value
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

	case td.OpReadAllProperties:
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

// ReadAction returns the latest cached action status.
func (m *ValueCacheService) ReadAction(thingID string, name string) (notif *msg.NotificationMessage) {

	notif = m.store.ReadAction(thingID, name)
	return notif
}

// ReadEvent returns the latest cached event value.
func (m *ValueCacheService) ReadEvent(thingID string, name string) (notif *msg.NotificationMessage) {

	notif = m.store.ReadEvent(thingID, name)
	return notif
}

// ReadProperty returns the last known cached notification of a property.
func (m *ValueCacheService) ReadProperty(thingID string, name string) (notif *msg.NotificationMessage) {

	notif = m.store.ReadProperty(thingID, name)
	return notif
}

// ReadMultipleProperties returns the value of cached properties.
// This returns a map of available values and a 'isCached' flag if all values are available.
// If not all requested values are available then isCached is false.
func (m *ValueCacheService) ReadMultipleProperties(
	thingID string, names []string) (v map[string]*msg.NotificationMessage, isCached bool) {

	propMap, isCached := m.store.ReadMultipleProperties(thingID, names)
	return propMap, isCached
}

// Start opens the logging destination.
func (m *ValueCacheService) Start() (err error) {
	return err
}

// Stop closes the logging destination.
func (m *ValueCacheService) Stop() {
}

// WriteEvent updates the event in the vcache
func (m *ValueCacheService) WriteAction(req *msg.RequestMessage) {
	notif := msg.NewNotificationMessage(
		req.SenderID, msg.AffordanceTypeAction, req.ThingID, req.Name, req.Input)
	notif.Timestamp = req.Timestamp
	notif.CorrelationID = req.CorrelationID
	m.store.WriteValue(notif)
}

// WriteEvent updates the event in the vcache
func (m *ValueCacheService) WriteEvent(notif *msg.NotificationMessage) {
	m.store.WriteValue(notif)
}

// WriteProperty updates the property in the vcache
func (m *ValueCacheService) WriteProperty(notif *msg.NotificationMessage) {
	m.store.WriteValue(notif)
}

// Create a new instance of the value cache module.
func NewValueCacheService() *ValueCacheService {

	thingID := vcacheapi.ValueCacheModuleType + "-" + shortid.MustGenerate()
	m := &ValueCacheService{
		HiveModuleBase: modules.NewHiveModuleBase(thingID, 0),
		store:          *NewVCacheStore(),
	}
	var _ vcacheapi.IValueCacheService = m // interface check
	return m
}
