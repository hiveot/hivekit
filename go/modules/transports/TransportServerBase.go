package transports

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/hiveot/hivekit/go/msg"
)

// Module properties that can be exposed in a TM
const PropName_NrConnections = "nrConnections"

// TransportServerBase implements the boilerplate of running a transport server module
// as defined in ITransportServer
// - Implemenets IHiveModule so it can act as a sink itself.
// - Manage incoming connections - see also ConnectionBase
// - Send requests, responses and notifications to connected clients
// - Aggregate messages from connections and send to Sink
//
// To initialize: call Init(moduleID, sink, connectURL)
type TransportServerBase struct {
	// moduleID/thingID is the unique instance ID of this server module.
	moduleID string

	// connections by clcid = {clientID}:{connectionID}
	connectionsByClcid map[string]IConnection

	// connectionIDs by clientID
	connectionsByClientID map[string][]string

	// The connection URL for this transport. Must be set during init
	connectURL string

	// mutex to manage the connections
	cmux sync.RWMutex

	// Request and Response channel helper.
	// Since some transports use unidirectional channels, a request to one channel
	// will result in a response over the other. RnRChan will pass the response from
	// one channel to the requester.
	RnrChan *msg.RnRChan

	// Sink for forwarding notifications
	notificationSink msg.NotificationHandler

	// Sink for forwarding requests
	requestSink msg.RequestHandler
}

// AddConnection adds a new connection and notifies subscribers.
// This requires the connection to have a unique client connection ID (connectionID).
//
// If an endpoint with this connectionID exists the existing connection is forcibly closed.
func (m *TransportServerBase) AddConnection(c IConnection) error {
	m.cmux.Lock()
	defer m.cmux.Unlock()

	if m.connectionsByClcid == nil {
		m.connectionsByClcid = make(map[string]IConnection)
	}
	if m.connectionsByClientID == nil {
		m.connectionsByClientID = make(map[string][]string)
	}

	// cinfo := c.GetConnectionInfo()
	clientID := c.GetClientID()
	cid := c.GetConnectionID()
	// the client's connectionID for lookup
	clcid := clientID + ":" + cid

	// Refuse this if an existing connection with this ID exist
	existingConn := m.connectionsByClcid[clcid]
	if existingConn != nil {
		err := fmt.Errorf("AddConnection. The connection ID '%s' of client '%s' already exists",
			cid, clientID)
		slog.Error("AddConnection: duplicate ConnectionID", "connectionID",
			cid, "err", err.Error())
		// close the existing connection
		m.removeConnection(existingConn)
		existingConn = nil
	}
	m.connectionsByClcid[clcid] = c
	// update the client index
	clientList := m.connectionsByClientID[clientID]
	if clientList == nil {
		clientList = []string{cid}
	} else {
		clientList = append(clientList, cid)
	}
	m.connectionsByClientID[clientID] = clientList
	// todo: nr of connections is a property of the module
	// m.UpdateProperty(PropName_NrConnections, len(m.connectionsByClcid))
	return nil
}

// CloseAllClientConnections closes all connections of the given client.
func (m *TransportServerBase) CloseAllClientConnections(clientID string) {
	m.cmux.Lock()
	defer m.cmux.Unlock()

	if m.connectionsByClientID == nil {
		return
	}

	cList := m.connectionsByClientID[clientID]
	for _, cid := range cList {
		// force-close the connection
		clcid := clientID + ":" + cid
		c := m.connectionsByClcid[clcid]
		if c != nil {
			delete(m.connectionsByClcid, clcid)
			c.Close()
		}
	}
	delete(m.connectionsByClientID, clientID)
	// todo: property for nr of connection
	//m.UpdateProperty(PropName_NrConnections, len(m.connectionsByClcid))
}

// CloseAll force-closes all connections
func (m *TransportServerBase) CloseAll() {
	m.cmux.Lock()
	defer m.cmux.Unlock()

	slog.Info("CloseAll. Closing remaining connections", "count", len(m.connectionsByClcid))
	for clcid, c := range m.connectionsByClcid {
		_ = clcid
		c.Close()
	}
	m.connectionsByClcid = nil
	m.connectionsByClientID = nil
	// todo: nr of connections
	//m.UpdateProperty(PropName_NrConnections, 0)
}

// ForEachConnection invoke handler for each client connection
// Intended for publishing event and property updates to subscribers
//
// This is concurrent safe as the iteration takes place on a copy.
// The handler can be blocking on non-blocking (goroutine)
func (m *TransportServerBase) ForEachConnection(handler func(c IConnection)) {
	// collect a list of connections
	m.cmux.Lock()
	connList := make([]IConnection, 0, len(m.connectionsByClcid))
	for _, c := range m.connectionsByClcid {
		connList = append(connList, c)
	}
	m.cmux.Unlock()
	//
	for _, c := range connList {
		// handler
		handler(c)
	}
}

// ForwardNotification passes notifications received by a server to the notification sink.
// This can be a service running on the server that has subscribed to a remote producer.
func (m *TransportServerBase) ForwardNotification(notif *msg.NotificationMessage) {
	if m.notificationSink == nil {
		// Receiving notifications but with no sink set so likely a wiring issue.
		slog.Error("ForwardNotification: no notification sink set. Server is not properly set up.",
			"module", m.moduleID,
			"operation", notif.Operation,
			"name", notif.Name,
		)
		return
	}
	m.notificationSink(notif)
}

// ForwardRequest passes a request from a client (agent) to the server request sink.
// HandleRequest method.
//
// This is used as the request handler of requests from incoming connections.
// If no sink os configured this returns an error
func (m *TransportServerBase) ForwardRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if m.requestSink == nil {
		slog.Error("ForwardRequest. Server has no request sink. Server is not properly set up.")
		return fmt.Errorf("ForwardRequest: no sink for request '%s/%s' to thingID '%s'",
			req.Operation, req.Name, req.ThingID)
	}
	err = m.requestSink(req, replyTo)
	return err
}

// GetConnectURL returns SSE connection URL of the server
// This uses the custom 'ssesc' schema which is non-wot compatible.
func (m *TransportServerBase) GetConnectURL() string {
	return m.connectURL
}

// GetConnectionByConnectionID locates the connection of the client using the client's connectionID
// This returns nil if no connection was found with the given connectionID
func (m *TransportServerBase) GetConnectionByConnectionID(clientID, connectionID string) (c IConnection) {
	clcid := clientID + ":" + connectionID
	m.cmux.Lock()
	defer m.cmux.Unlock()

	if m.connectionsByClcid == nil {
		return nil
	}
	c = m.connectionsByClcid[clcid]
	return c
}

// GetConnectionByClientID locates the first connection of the client using its account ID.
// Intended to find agents which only have a single connection.
// This returns nil if no connection was found with the given login
func (m *TransportServerBase) GetConnectionByClientID(clientID string) (c IConnection) {

	m.cmux.Lock()
	defer m.cmux.Unlock()
	if m.connectionsByClientID == nil {
		// no incoming connections yet
		slog.Warn("Requesting connection for client but none have been received", "clientID", clientID)
		return nil
	}
	cList := m.connectionsByClientID[clientID]
	if len(cList) == 0 {
		return nil
	}
	clcid := clientID + ":" + cList[0]

	// return the first connection of this client
	c = m.connectionsByClcid[clcid]
	if c == nil {
		slog.Error("GetConnectionByClientID: the client's connection list has disconnected endpoints",
			"clientID", clientID, "nr alleged connections", len(cList))
	}
	return c
}

// GetModuleID returns the module's Thing ID
func (m *TransportServerBase) GetModuleID() string {
	return m.moduleID
}

// GetTM returns the module's TM describing its properties, actions and events.
// This server does not expose a TM.
func (m *TransportServerBase) GetTM() string {
	return ""
}

// Initialize the module base with a moduleID and a messaging sink
//
//	moduleID is the transport instance ID to identify as.
//	connectURL is the URL this module can be reached at.
func (m *TransportServerBase) Init(moduleID string, connectURL string) {
	m.moduleID = moduleID
	m.connectURL = connectURL
	// m.HiveModuleBase.Init(moduleID, sink)
	m.RnrChan = msg.NewRnRChan()
}

// // onNotificationFromSink receives an incoming notification from the registered sink.
// //
// // This sends the notifications to subscribed connections
// func (m *TransportServerBase) onNotificationFromSink(notif *msg.NotificationMessage) {
// 	// the reason for the extra indirection is to ensure we're receiving the notification
// 	// independently from how it is processed. Primarily useful in debugging.
// 	m.SendNotification(notif)
// }

// // onNotificationFromConnection receives an incoming notification from the remote connection
// //
// // This sends the notifications to the consumer notification handler, if any.
// func (m *TransportServerBase) onNotificationFromConnection(notif *msg.NotificationMessage) {
// 	// the reason for the extra indirection is to ensure we're receiving the notification
// 	// independently from whether a consumer has set one.
// 	m.ForwardNotification(notif)
// }

// removeConnection removes the connection.
// non-concurrent safe internal function that can be used from a locked section.
// This will close the connnection if it isn't closed already.
// Call this after the connection is closed or before closing.
func (m *TransportServerBase) removeConnection(c IConnection) {

	clientID := c.GetClientID()
	connectionID := c.GetConnectionID()
	clcid := clientID + ":" + connectionID

	// if nothing to do
	if m.connectionsByClcid == nil {
		slog.Warn("RemoveConnection: no connections remaining",
			"clcid", clcid)
		return
	}

	existingConn := m.connectionsByClcid[clcid]
	// force close the existing connection just in case
	if existingConn != nil {
		//clientID = existingConn.GetClientID()
		existingConn.Close()
		delete(m.connectionsByClcid, clcid)
	} else if len(m.connectionsByClcid) > 0 {
		// this is unexpected. Not all connections were closed but this one is gone.
		slog.Warn("RemoveConnection: connectionID not found",
			"clcid", clcid)
		return
	}
	// remove the cid from the client connection list
	clientCids := m.connectionsByClientID[clientID]
	i := slices.Index(clientCids, connectionID)
	if i < 0 {
		slog.Info("RemoveConnection: existing connection not in the connectionID list. Was it forcefully removed?",
			"clientID", clientID, "connectionID", connectionID)

		// TODO: considering the impact of this going wrong, is it better to recover?
		// A: delete the bad entry and try the next connection
		// B: close all client connections

	} else {
		clientCids = slices.Delete(clientCids, i, i+1)
		//clientCids = utils.Remove(clientCids, i)
		m.connectionsByClientID[clientID] = clientCids
	}
	// todo: module properties
	// m.UpdateProperty(PropName_NrConnections, len(m.connectionsByClcid))
}

// RemoveConnection removes the connection by its connectionID
// This will close the connnection if it isn't closed already.
// Call this after the connection is closed or before closing.
func (m *TransportServerBase) RemoveConnection(c IConnection) {
	m.cmux.Lock()
	defer m.cmux.Unlock()
	m.removeConnection(c)
}

// SendNotification [agent] server sends a notification to its connections
// The connection handles subscriptions.
func (m *TransportServerBase) SendNotification(notif *msg.NotificationMessage) {
	m.ForEachConnection(func(c IConnection) {
		c.SendNotification(notif)
	})
}

// SendRequest [consumer] sends a request over the connection to an agent.
//
// agentID is the agent's authentication ID that hosts one or more Things.
// req is the request message envelope to send
// replyTo is the callback handler, or nil to handle replies via the async
// module callback, which by default is the sink's HandleResponse.
//
// Note that the request message contains the ThingID of the thing for which the request
// is intended. The agent must know how to forward the request to the Thing.
func (m *TransportServerBase) SendRequest(
	agentID string, req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	c := m.GetConnectionByClientID(agentID)
	if c == nil {
		return fmt.Errorf("No connection with agent '%s'", agentID)
	}
	err = c.SendRequest(req, replyTo)
	return err
}

// SendResponse [agent] sends the response message over the transport to a remote
// consumer with the given client and connection ID.
// This is equivalent to calling SendResponse on the connection itself.
//
//	clientID identifies the consumer to send the response to
func (m *TransportServerBase) SendResponse(
	clientID, cid string, resp *msg.ResponseMessage) (err error) {
	// var c IConnection

	c := m.GetConnectionByConnectionID(clientID, cid)

	// if nothing to do
	if c == nil {
		err = fmt.Errorf("SendResponse: connection for clientID '%s' and connectionID '%s' not found", clientID, cid)
		return err
	}
	err = c.SendResponse(resp)
	return err
}

// Set the handler that will receive notifications emitted by this module
func (m *TransportServerBase) SetNotificationSink(consumer msg.NotificationHandler) {
	m.notificationSink = consumer
}

// Set the handler that will receive notifications emitted by this module
func (m *TransportServerBase) SetRequestSink(sink msg.RequestHandler) {
	m.requestSink = sink
}
