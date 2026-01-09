package transports

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/msg"
)

// Module properties that can be exposed in a TM
const PropName_NrConnections = "nrConnections"

// TransportModuleBase implements the boilerplate of running a transport module
// as defined in ITransportModule
// - HiveModuleBase
// - Manage incoming connections - see also ConnectionBase
// - Send requests, responses and notifications to connected clients
// - Aggregate messages from connections and send to Sink
//
// To initialize: call Init(moduleID, sink, connectURL)
type TransportModuleBase struct {
	modules.HiveModuleBase

	// connections by clcid = {clientID}:{connectionID}
	connectionsByClcid map[string]IServerConnection

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
	RnrChan *RnRChan
}

// AddConnection adds a new connection and notifies subscribers.
// This requires the connection to have a unique client connection ID (connectionID).
//
// If an endpoint with this connectionID exists the existing connection is forcibly closed.
func (m *TransportModuleBase) AddConnection(c IServerConnection) error {
	m.cmux.Lock()
	defer m.cmux.Unlock()

	if m.connectionsByClcid == nil {
		m.connectionsByClcid = make(map[string]IServerConnection)
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
	// nr of connections is a property of the module
	m.UpdateProperty(PropName_NrConnections, len(m.connectionsByClcid))
	return nil
}

// CloseAllClientConnections closes all connections of the given client.
func (m *TransportModuleBase) CloseAllClientConnections(clientID string) {
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
	m.UpdateProperty(PropName_NrConnections, len(m.connectionsByClcid))
}

// CloseAll force-closes all connections
func (m *TransportModuleBase) CloseAll() {
	m.cmux.Lock()
	defer m.cmux.Unlock()

	slog.Info("CloseAll. Closing remaining connections", "count", len(m.connectionsByClcid))
	for clcid, c := range m.connectionsByClcid {
		_ = clcid
		c.Close()
	}
	m.connectionsByClcid = nil
	m.connectionsByClientID = nil
	m.UpdateProperty(PropName_NrConnections, 0)
}

// ForEachConnection invoke handler for each client connection
// Intended for publishing event and property updates to subscribers
//
// This is concurrent safe as the iteration takes place on a copy.
// The handler can be blocking on non-blocking (goroutine)
func (m *TransportModuleBase) ForEachConnection(handler func(c IServerConnection)) {
	// collect a list of connections
	m.cmux.Lock()
	connList := make([]IServerConnection, 0, len(m.connectionsByClcid))
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

// GetConnectURL returns SSE connection URL of the server
// This uses the custom 'ssesc' schema which is non-wot compatible.
func (m *TransportModuleBase) GetConnectURL() string {
	return m.connectURL
}

// GetConnectionByConnectionID locates the connection of the client using the client's connectionID
// This returns nil if no connection was found with the given connectionID
func (m *TransportModuleBase) GetConnectionByConnectionID(clientID, connectionID string) (c IServerConnection) {
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
func (m *TransportModuleBase) GetConnectionByClientID(clientID string) (c IServerConnection) {

	m.cmux.Lock()
	defer m.cmux.Unlock()
	if m.connectionsByClientID == nil {
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

// Initialize the module base with a moduleID and a messaging sink
func (m *TransportModuleBase) Init(moduleID string, sink modules.IHiveModule, connectURL string) {
	m.connectURL = connectURL
	m.HiveModuleBase.Init(moduleID, sink)
}

// removeConnection removes the connection.
// non-concurrent safe internal function that can be used from a locked section.
// This will close the connnection if it isn't closed already.
// Call this after the connection is closed or before closing.
func (m *TransportModuleBase) removeConnection(c IServerConnection) {
	// cinfo := c.GetConnectionInfo()
	clientID := c.GetClientID()
	connectionID := c.GetConnectionID()
	clcid := clientID + ":" + connectionID

	m.cmux.Lock()
	defer m.cmux.Unlock()

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
	m.UpdateProperty(PropName_NrConnections, len(m.connectionsByClcid))
}

// RemoveConnection removes the connection by its connectionID
// This will close the connnection if it isn't closed already.
// Call this after the connection is closed or before closing.
func (m *TransportModuleBase) RemoveConnection(c IServerConnection) {
	m.cmux.Lock()
	defer m.cmux.Unlock()
	m.removeConnection(c)
}

// SendNotification [agent] sends a notification to all connections.
// The connection handles subscriptions.
func (m *TransportModuleBase) SendNotification(notif *msg.NotificationMessage) {
	m.ForEachConnection(func(c IServerConnection) {
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
func (m *TransportModuleBase) SendRequest(
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
func (m *TransportModuleBase) SendResponse(
	clientID, cid string, resp *msg.ResponseMessage) (err error) {
	// var c IServerConnection

	c := m.GetConnectionByConnectionID(clientID, cid)

	// if nothing to do
	if c == nil {
		err = fmt.Errorf("SendResponse: connection for clientID '%s' and connectionID '%s' not found", clientID, cid)
		return err
	}
	err = c.SendResponse(resp)
	return err
}
