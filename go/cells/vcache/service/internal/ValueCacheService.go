package internal

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/vcache"
	vcacheapi "github.com/hiveot/hivekit/go/cells/vcache"
	"github.com/hiveot/hivekit/go/utils"
)

// ValueCacheService is the value-cache service implementation
// this implements the IVCache and IHiveCell interface
//
// This service stores that latest value of property and event notifications.
type ValueCacheService struct {
	*cells.HiveCellBase

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
	m.WriteValue(notif)
	// forward the notification up the chain
	m.ForwardNotification(notif)
}

// HandleRequest handles requests for use by this service.
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
	var found bool
	var output any

	// handle read requests
	switch req.Operation {
	// wot doesnt define operations for reading events
	case wot.HTOpReadEvent:
		// this is not a wot defined operation
		// return the notification itself because the time of the event is relevant
		output, found = m.ReadEvent(req.ThingID, req.Name)
	case wot.OpReadProperty:
		// The protocol binding converts the output from ThingValue to whatever is specified
		output, found = m.ReadProperty(req.ThingID, req.Name)

	case wot.OpReadMultipleProperties:
		// The protocol binding converts the output from ThingValue to whatever is specified
		var names []string
		err := utils.DecodeAsObject(req.Input, &names)
		if err != nil {
			// missing names
			slog.Warn("HandleRequest: ReadMultipleProperties, missing names. Forwarding the request",
				"senderID", req.SenderID, "thingID", req.ThingID)
		} else {
			output, found = m.ReadMultipleProperties(req.ThingID, names)
		}

	case wot.OpReadAllProperties:
		// querying all properties is not supported as there is no knowledge of all possible
		// properties.
		found = false
	}

	if !found {
		// forward the request to the actual device
		return m.ForwardRequest(req, replyTo)
	}

	// a cached notification was found
	resp := req.CreateResponse(output, nil)
	return replyTo(resp)
}

// ReadAction returns the latest cached action status.
func (m *ValueCacheService) ReadAction(thingID string, name string) (as msg.ActionStatus, found bool) {

	return m.store.ReadAction(thingID, name)
}

// ReadEvent returns the latest cached event value.
func (m *ValueCacheService) ReadEvent(thingID string, name string) (tv msg.ThingValue, found bool) {

	return m.store.ReadEvent(thingID, name)
}

// ReadProperty returns the last known cached notification of a property.
func (m *ValueCacheService) ReadProperty(thingID string, name string) (tv msg.ThingValue, found bool) {

	return m.store.ReadProperty(thingID, name)
}

// ReadMultipleProperties returns the value of cached properties.
// This returns a map of available values and a 'isCached' flag if all values are available.
// If not all requested values are available then isCached is false.
func (m *ValueCacheService) ReadMultipleProperties(
	thingID string, names []string) (v map[string]msg.ThingValue, isCached bool) {

	propMap, isCached := m.store.ReadMultipleProperties(thingID, names)
	return propMap, isCached
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

// WriteValue updates the propery, event or action in the vcache
func (m *ValueCacheService) WriteValue(notif *msg.NotificationMessage) {
	m.store.WriteValue(notif)
}

// NewValueCacheService creates a ready-to-use instance of the value cache service.
func NewValueCacheService() (*ValueCacheService, error) {

	thingID := vcacheapi.ValueCacheDefaultThingID
	svc := &ValueCacheService{
		HiveCellBase: cells.NewHiveCellBase(thingID),
		store:        *NewVCacheStore(),
	}
	var _ vcacheapi.IValueCacheService = svc // interface check
	return svc, nil
}
