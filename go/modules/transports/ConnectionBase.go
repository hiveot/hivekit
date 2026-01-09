package transports

import (
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
)

// ConnectionBase is a base type for implementing transport connections.
// Use of this is totally optional.
// This implements the IServerConnection interface.
type ConnectionBase struct {
	// Authenticated ID of the client
	ClientID string

	// The connection identification
	ConnectionID string

	// // Connection information such as clientID, cid, address, protocol etc
	// cinfo ConnectionInfo

	remoteAddr    string
	observations  Subscriptions
	subscriptions Subscriptions

	// SendNotificationHandler msg.NotificationHandler
	// connections clients are asynchronous
	// SendRequestHandler func(req *msg.RequestMessage) (err error)
	//
	// SendResponseHandler msg.ResponseHandler
	isConnected atomic.Bool

	Mux sync.RWMutex
}

func (c *ConnectionBase) Disconnect() {
	c.isConnected.Store(false)
}

func (c *ConnectionBase) GetClientID() string {
	return c.ClientID
}

func (c *ConnectionBase) GetConnectionID() string {
	return c.ConnectionID
}
func (c *ConnectionBase) IsConnected() bool {
	return c.isConnected.Load()
}

// func (c *ConnectionBase) SendNotification(msg *msg.NotificationMessage) error {
// 	c.Mux.Lock()
// 	h := c.SendNotificationHandler
// 	c.Mux.Unlock()
// 	if h != nil {
// 		h(msg)
// 	}
// 	return nil
// }

// func (c *ConnectionBase) SendRequest(msg *msg.RequestMessage) error {
// 	c.Mux.Lock()
// 	h := c.SendRequestHandler
// 	c.Mux.Unlock()

// 	if h != nil && c.observations.IsSubscribed(msg.ThingID, msg.Name) {
// 		return h(msg)
// 	}
// 	return fmt.Errorf("no request sender set")
// }

// func (c *ConnectionBase) SendResponse(resp *msg.ResponseMessage) error {
// 	c.Mux.Lock()
// 	h := c.SendResponseHandler
// 	c.Mux.Unlock()

// 	if h != nil {
// 		return h(resp)
// 	}
// 	return nil
// }

func (c *ConnectionBase) SetConnectHandler(h ConnectionHandler) {
	c.Mux.Lock()
	defer c.Mux.Unlock()
}

// SetNotificationHandler is ignored as this is an outgoing 1-way connection
// func (c *ConnectionBase) SetNotificationHandler(h msg.NotificationHandler) {
// 	c.Mux.Lock()
// 	defer c.Mux.Unlock()
// 	c.SendNotificationHandler = h
// }

// SetRequestHandler is ignored as this is an outgoing 1-way connection
// func (c *ConnectionBase) SetRequestHandler(h msg.RequestHandler) {
// }

// // SetResponseHandler is ignored as this is an outgoing 1-way connection
// func (c *ConnectionBase) SetResponseHandler(h msg.ResponseHandler) {
// }

// HasSubscription returns true if this connection has subscribed to the given notification
func (sc *ConnectionBase) HasSubscription(notif *msg.NotificationMessage) bool {
	switch notif.Operation {

	case wot.OpSubscribeEvent, wot.OpSubscribeAllEvents:
		correlationID := sc.subscriptions.GetSubscription(notif.ThingID, notif.Name)
		if correlationID != "" {
			slog.Info("HasSubscription (event subscription)",
				slog.String("clientID", sc.ClientID),
				slog.String("thingID", notif.ThingID),
				slog.String("event name", notif.Name),
			)
			return true
		}
	case wot.OpObserveProperty, wot.OpObserveMultipleProperties, wot.OpObserveAllProperties:
		correlationID := sc.observations.GetSubscription(notif.ThingID, notif.Name)
		if correlationID != "" {
			slog.Info("HasSubscription (observed property(ies))",
				slog.String("clientID", sc.ClientID),
				slog.String("thingID", notif.ThingID),
				slog.String("name", notif.Name),
			)
			return true
		}
	case wot.OpInvokeAction:
		// action progress update, for original sender only
		slog.Info("HasSubscription (action status)",
			slog.String("clientID", sc.ClientID),
			slog.String("thingID", notif.ThingID),
			slog.String("name", notif.Name),
		)
		return true
	default:
		slog.Warn("Unknown notification: " + notif.Operation)
	}
	return false
}

// Initialize the connection base. Call this before use.
func (c *ConnectionBase) Init(clientID, remoteAddr, cid string) {
	c.ClientID = clientID
	c.ConnectionID = cid
	c.remoteAddr = remoteAddr
	c.isConnected.Store(true)
}

// Subscribe to an event.
func (c *ConnectionBase) SubscribeEvent(dThingID, name string, corrID string) {
	c.subscriptions.Subscribe(dThingID, name, corrID)
}
func (c *ConnectionBase) ObserveProperty(dThingID, name string, corrID string) {
	c.observations.Subscribe(dThingID, name, corrID)
}
func (c *ConnectionBase) UnsubscribeEvent(dThingID, name string) {
	c.subscriptions.Unsubscribe(dThingID, name)
}
func (c *ConnectionBase) UnobserveProperty(dThingID, name string) {
	c.observations.Unsubscribe(dThingID, name)
}

//func (c *DummyConnection) WriteProperty(thingID, name string, value any, correlationID string, senderID string) (status string, err error) {
//	return "", nil
//}

// Subscriptions manages event/property subscriptions of a consumer connection.
//
// This uses "+" as wildcards
type Subscriptions struct {

	// map of subscriptions to correlationID
	// subscriptions of this connection in the form {dThingID}.{name}
	// not many are expected.
	subscriptions map[string]string

	// mutex for access to subscriptions
	mux sync.RWMutex
}

// GetSubscription returns the correlation ID if this client session has subscribed to
// events or properties from the Thing and name.
// If the subscription is unknown then return an empty string.
func (s *Subscriptions) GetSubscription(thingID string, name string) string {
	s.mux.RLock()
	defer s.mux.RUnlock()

	if len(s.subscriptions) == 0 {
		return ""
	}
	// wildcards
	thingWC := "+." + name
	nameWC := thingID + ".+"
	sub := thingID + "." + name
	for k, v := range s.subscriptions {
		if k == "+.+" {
			// step 1, full wildcard subscriptions
			return v
		} else if k == thingWC || k == nameWC {
			// step 1, thing or name wildcard subscriptions
			return v
		} else if k == sub {
			// step 1, exact match subscriptions
			return v
		}
	}
	return ""
}

// IsSubscribed returns true  if this client session has subscribed to
// events or properties from the Thing and name
func (s *Subscriptions) IsSubscribed(thingID string, name string) bool {
	corrID := s.GetSubscription(thingID, name)
	return corrID != ""
}

// Subscribe adds a subscription for a thing event/property
func (s *Subscriptions) Subscribe(thingID string, name string, correlationID string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	if thingID == "" {
		thingID = "+"
	}
	if name == "" {
		name = "+"
	}
	subKey := thingID + "." + name
	if s.subscriptions == nil {
		s.subscriptions = make(map[string]string)
	}
	s.subscriptions[subKey] = correlationID
}

// Unsubscribe removes a subscription for a thing event/property
func (s *Subscriptions) Unsubscribe(dThingID string, name string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	if dThingID == "" {
		dThingID = "+"
	}
	if name == "" {
		name = "+"
	}
	subKey := dThingID + "." + name
	delete(s.subscriptions, subKey)
}
