package module

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
)

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
//
// The result of WoT operations follows the websocket message pattern of returning the value.
// In case of reading evwents however this returns the notification itself as the timestamp
// is important.
func (m *VCacheModule) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var isCached bool
	var value any

	// handle read requests
	switch req.Operation {
	// wot doesnt define operations for reading events
	case wot.HTOpReadEvent:
		// this is not a wot defined operation
		// return the notification itself because the time of the event is relevant
		notif := m.ReadEvent(req.ThingID, req.Name)
		if notif != nil {
			isCached = true
			value = notif //.Data
		}
	case wot.OpReadProperty:
		// WoT specifies that read property returns the latest value
		notif := m.ReadProperty(req.ThingID, req.Name)
		if notif != nil {
			isCached = true
			value = notif.Data
		}
	case wot.OpReadMultipleProperties:
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
