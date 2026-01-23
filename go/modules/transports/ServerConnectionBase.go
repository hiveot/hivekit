package transports

import (
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
)

// ServerConnectionBase is a base type for implementing transport connections server side.
// Use of this is totally optional.
// This implements the IConnection interface.
type ServerConnectionBase struct {
	// Authenticated ID of the remote client
	ClientID string

	// The connection identification
	ConnectionID string

	// ConnectHandler is the handler to invoke when a connection status changes
	ConnectHandler ConnectionHandler

	// Remote address of the connection
	remoteAddr string

	// property observations made by the client
	observations Subscriptions

	// event subscription made by the client
	subscriptions Subscriptions

	// SendNotificationHandler msg.NotificationHandler
	// connections clients are asynchronous
	// SendRequestHandler func(req *msg.RequestMessage) (err error)
	//
	// SendResponseHandler msg.ResponseHandler
	isConnected atomic.Bool

	// Mux to update subscriptions, connection status
	Mux sync.RWMutex

	// Connection sink that handles the incoming messages
	Sink modules.IHiveModule
}

// ConnectWithToken is defined in IConnection and does not apply to server side connections.
// (not yet anyways)
func (c *ServerConnectionBase) ConnectWithToken(clientID, token string) error {
	return errors.New("Not for server connections")
}

func (c *ServerConnectionBase) Disconnect() {
	c.isConnected.Store(false)
}

func (c *ServerConnectionBase) GetClientID() string {
	return c.ClientID
}

func (c *ServerConnectionBase) GetConnectionID() string {
	return c.ConnectionID
}

// HasSubscription returns true if this connection has subscribed to the given
// event notification or observing property changes.
func (sc *ServerConnectionBase) HasSubscription(notif *msg.NotificationMessage) bool {
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
func (c *ServerConnectionBase) Init(clientID, remoteAddr, cid string, sink modules.IHiveModule) {
	c.ClientID = clientID
	c.ConnectionID = cid
	c.remoteAddr = remoteAddr
	c.isConnected.Store(true)
	c.Sink = sink
}

func (c *ServerConnectionBase) IsConnected() bool {
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

func (c *ServerConnectionBase) SetConnectHandler(h ConnectionHandler) {
	c.Mux.Lock()
	defer c.Mux.Unlock()
	c.ConnectHandler = h
}

// Set the destination sink for received messages
func (c *ServerConnectionBase) SetSink(sink modules.IHiveModule) {
	c.Mux.Lock()
	defer c.Mux.Unlock()
	c.Sink = sink
}

// Subscribe to an event.
func (c *ServerConnectionBase) SubscribeEvent(dThingID, name string, corrID string) {
	c.subscriptions.Subscribe(dThingID, name, corrID)
}
func (c *ServerConnectionBase) ObserveProperty(dThingID, name string, corrID string) {
	c.observations.Subscribe(dThingID, name, corrID)
}
func (c *ServerConnectionBase) UnsubscribeEvent(dThingID, name string) {
	c.subscriptions.Unsubscribe(dThingID, name)
}
func (c *ServerConnectionBase) UnobserveProperty(dThingID, name string) {
	c.observations.Unsubscribe(dThingID, name)
}

//func (c *DummyConnection) WriteProperty(thingID, name string, value any, correlationID string, senderID string) (status string, err error) {
//	return "", nil
//}
