package transport

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"github.com/hiveot/hivekit/go/api/msg"
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

	// appRequestHook is the application handler of requests addressed to this module's thingID.
	//
	// HandleRequest will invoke this callback or forward requests not destined for
	// this module (moduleID != request.ThingID) to requestSink.
	appRequestHook msg.RequestHandler

	// authenticator for incoming connections and for adding form security info
	authenticator IAuthenticator

	// The base URL used to connect. Used to set TD.Base field when adding forms
	connectURL string

	// connections by clcid = {clientID}:{connectionID}
	connectionsByClcid map[string]IConnection

	// connectionIDs by clientID
	connectionsByClientID map[string][]string

	// mutex to manage the connections
	cmux sync.RWMutex

	// Sink for forwarding notifications
	notificationSink msg.NotificationHandler

	// Sink for forwarding requests
	requestSink msg.RequestHandler

	// Request and Response channel helper.
	// Since some transports use unidirectional channels, a request to one channel
	// will result in a response over the other. RnRChan will pass the response from
	// one channel to the requester.
	RnrChan *msg.RnRChan

	// the server module thingID used for sending connect/disconnect notifications
	thingID string
}

// AddConnection adds a new connection and notifies subscribers with a ServerConnectEvent notification.
//
// The connection can be looked up with GetConnectionByClientID or indirectly
// using DetermineAgentConnection(thingID).
//
// The given connection is stored under clientID:connectionID. If the connectionID
// is empty then only a single connection for the client can be used.
//
// If an endpoint with this clientID:connectionID exists the existing connection is forcibly closed.
func (srv *TransportServerBase) AddConnection(c IConnection) error {
	var clientID string
	var cid string
	// enter protected block
	prot := func() {
		srv.cmux.Lock()
		defer srv.cmux.Unlock()

		if srv.connectionsByClcid == nil {
			srv.connectionsByClcid = make(map[string]IConnection)
		}
		if srv.connectionsByClientID == nil {
			srv.connectionsByClientID = make(map[string][]string)
		}

		clientID = c.GetClientID()
		cid = c.GetConnectionID()
		// the client's connectionID for lookup
		clcid := clientID + ":" + cid

		// Refuse this if an existing connection with this ID exist
		existingConn := srv.connectionsByClcid[clcid]
		if existingConn != nil {
			err := fmt.Errorf("AddConnection. The connection ID '%s' of client '%s' already exists",
				cid, clientID)
			slog.Error("AddConnection: duplicate ConnectionID", "connectionID",
				cid, "err", err.Error())
			// close the existing connection
			srv.removeConnection(existingConn)
			existingConn = nil
		}
		srv.connectionsByClcid[clcid] = c
		// update the client index
		clientList := srv.connectionsByClientID[clientID]
		if clientList == nil {
			clientList = []string{cid}
		} else {
			clientList = append(clientList, cid)
		}
		srv.connectionsByClientID[clientID] = clientList
	}
	prot()

	// notify listeners outside of locked area
	// publish a notification about the new connection
	connectionInfo := ConnectionInfo{
		ClientID:     clientID,
		ConnectionID: cid,
	}
	// publish a notification for those interested
	senderID := srv.thingID
	thingID := srv.thingID
	notif := msg.NewNotificationMessage(senderID, msg.AffordanceTypeEvent, thingID,
		ServerConnectEvent, connectionInfo)
	srv.ForwardNotification(notif)
	return nil
}

// CloseAllClientConnections closes all connections of the given client.
func (srv *TransportServerBase) CloseAllClientConnections(clientID string) {
	srv.cmux.Lock()
	defer srv.cmux.Unlock()

	if srv.connectionsByClientID == nil {
		return
	}

	cList := srv.connectionsByClientID[clientID]
	for _, cid := range cList {
		// force-close the connection
		clcid := clientID + ":" + cid
		c := srv.connectionsByClcid[clcid]
		if c != nil {
			delete(srv.connectionsByClcid, clcid)
			c.Close()
		}
	}
	delete(srv.connectionsByClientID, clientID)
	// todo: property for nr of connection
	//m.UpdateProperty(PropName_NrConnections, len(m.connectionsByClcid))
}

// CloseAll force-closes all connections
func (srv *TransportServerBase) CloseAll() {
	srv.cmux.Lock()
	defer srv.cmux.Unlock()

	slog.Info("CloseAll. Closing remaining connections", "count", len(srv.connectionsByClcid))
	for clcid, c := range srv.connectionsByClcid {
		_ = clcid
		c.Close()
	}
	srv.connectionsByClcid = nil
	srv.connectionsByClientID = nil
	// todo: nr of connections
	//m.UpdateProperty(PropName_NrConnections, 0)
}

// Get the agent/producer (reverse) connection that serves the given ThingID.
//
// Intended for looking up an agent with a reverse connection, when acting as a gateway.
// HiveOT agents that use reverse connections are required to add their agentID as a prefix
// in the thingID of the TDs they publish. For example: "agent1:thing1". This is a
// convention but it is not required by the WoT specifications.
func (m *TransportServerBase) DetermineAgentConnection(thingID string) (IConnection, error) {
	parts := strings.Split(thingID, ":")
	agentID := parts[0]

	c := m.GetConnectionByClientID(agentID)
	if c == nil {
		return nil, fmt.Errorf("No connection found for ThingID '%s'", thingID)
	}
	return c, nil
}

// ForEachConnection invoke handler for each client connection
// Intended for publishing event and property updates to subscribers
//
// This is concurrent safe as the iteration takes place on a copy.
// The handler can be blocking on non-blocking (goroutine)
func (srv *TransportServerBase) ForEachConnection(handler func(c IConnection)) {
	// collect a list of connections
	srv.cmux.Lock()
	connList := make([]IConnection, 0, len(srv.connectionsByClcid))
	for _, c := range srv.connectionsByClcid {
		connList = append(connList, c)
	}
	srv.cmux.Unlock()
	//
	for _, c := range connList {
		// handler
		handler(c)
	}
}

// ForwardNotification passes received notifications to the linked notification sink.
// These notifications are typically sent by remote agents that use RC, or by this server
// module itself to notify of connect/disconnects.  They are intended for services
// running on the server, or to be forwarded to clients that subscribed to them.
//
// This behavior differes from HiveModuleBase as the forwarded notifications are received from a remote
// client (agent) and must be forwarded to the server chain, which in turn passes it to subscribers.
//
// This logs a warning if no notification handler is set as the notification will be lost.
func (srv *TransportServerBase) ForwardNotification(notif *msg.NotificationMessage) {
	if srv.notificationSink == nil {
		// Receiving notifications but with no sink set so likely a wiring issue.
		// This can be intentional in testing.
		slog.Warn("ForwardNotification: no notification sink set. Server is not fully set up.",
			"module", srv.thingID,
			"affordance", notif.AffordanceType,
			"name", notif.Name,
		)
		return
	}
	srv.notificationSink(notif)
}

// ForwardRequest passes a request to the configured request sink.
//
// This is used as the request handler of requests from incoming connections
// or for requests passed down as part of the module chain.
//
// If no sink os configured this returns an error
func (srv *TransportServerBase) ForwardRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if srv.requestSink == nil {
		slog.Error("ForwardRequest. Server has no request sink. Server is not fully set up.",
			"op", req.Operation, "thingID", req.ThingID, "name", req.Name)
		return fmt.Errorf("ForwardRequest: no sink for request '%s/%s' to thingID '%s'",
			req.Operation, req.Name, req.ThingID)
	}
	err = srv.requestSink(req, replyTo)
	return err
}

// GetConnectURL returns connection URL of the server
// This is set with init
func (m *TransportServerBase) GetConnectURL() string {
	return m.connectURL
}

// GetConnectionByConnectionID locates the connection of the client using the client's connectionID
// This returns nil if no connection was found with the given connectionID
func (srv *TransportServerBase) GetConnectionByConnectionID(clientID, connectionID string) (c IConnection) {
	clcid := clientID + ":" + connectionID
	srv.cmux.Lock()
	defer srv.cmux.Unlock()

	if srv.connectionsByClcid == nil {
		return nil
	}
	c = srv.connectionsByClcid[clcid]
	return c
}

// GetConnectionByClientID locates the first connection of the client using its account ID.
// Intended to find agents which only have a single connection.
// This returns nil if no connection was found with the given login
func (srv *TransportServerBase) GetConnectionByClientID(clientID string) (c IConnection) {

	srv.cmux.Lock()
	defer srv.cmux.Unlock()
	if srv.connectionsByClientID == nil {
		// no incoming connections yet
		slog.Warn("Requesting connection for client but none have been received", "clientID", clientID)
		return nil
	}
	cList := srv.connectionsByClientID[clientID]
	if len(cList) == 0 {
		return nil
	}
	clcid := clientID + ":" + cList[0]

	// return the first connection of this client
	c = srv.connectionsByClcid[clcid]
	if c == nil {
		slog.Error("GetConnectionByClientID: the client's connection list has disconnected endpoints",
			"clientID", clientID, "nr alleged connections", len(cList))
	}
	return c
}

// Return the module instance ID
func (m *TransportServerBase) GetThingID() string {
	return m.thingID
}

// Handle a notification this module (or downstream in the chain) subscribed to.
// Notifications are forwarded to their upstream sink, which for a server is the
// client.
func (m *TransportServerBase) HandleNotification(notif *msg.NotificationMessage) {
	m.SendNotification(notif)
}

// HandleRequest sends requests to connected client.
//
// This only happens when a consumer on the server or gateway passes the request to
// this server module through the chain, when this server is the sink for the consumer.
// Transport modules forward requests to connected clients instead of processing them locally.
//
// This returns an error when the destination for the request cannot be found.
// If multiple server protocols are used it is okay to try them one by one.
//
// When using the router/directory module combo, this should not be used. Instead the router
// determines the destination using the TD in the directory and determines the agent and connection
// without relying on the agentID in ThingID convention.
func (m *TransportServerBase) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	if req.ThingID == m.thingID {
		if m.appRequestHook != nil {
			return m.appRequestHook(req, replyTo)
		} else {
			return fmt.Errorf("HandleRequest: no request handler set for this transport server module '%s'", m.thingID)
		}
	}

	// first attempt to procss the when targeted at this module
	// if the request is not for this module then pass it to the remote agent
	// if the agent isn't connected then this returns an error. This can be valid
	// in case multiple server protocols are used and the request is for another protocol.
	c, err := m.DetermineAgentConnection(req.ThingID)
	if err == nil {
		err = c.SendRequest(req, replyTo)
	}
	return err
}

// removeConnection removes the connection and sends an event notification.
// non-concurrent safe internal function that can be used from a locked section.
// This will close the connnection if it isn't closed already.
// Call this after the connection is closed or before closing.
func (srv *TransportServerBase) removeConnection(c IConnection) {
	clientID := c.GetClientID()
	connectionID := c.GetConnectionID()
	clcid := clientID + ":" + connectionID

	// if nothing to do
	if srv.connectionsByClcid == nil {
		// Most likely caused by a call to CloseAll() before the clients shut down.
		// this isn't very nice but lets handle it gracefull.y
		slog.Warn("RemoveConnection: connection was already removed",
			"clcid", clcid)
		return
	}

	existingConn := srv.connectionsByClcid[clcid]
	// force close the existing connection just in case
	if existingConn != nil {
		//clientID = existingConn.GetClientID()
		existingConn.Close()
		delete(srv.connectionsByClcid, clcid)
	} else if len(srv.connectionsByClcid) > 0 {
		// this is unexpected. Not all connections were closed but this one is gone.
		slog.Error("RemoveConnection: connectionID not found",
			"clcid", clcid)
		return
	}
	// remove the cid from the client connection list
	clientCids := srv.connectionsByClientID[clientID]
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
		srv.connectionsByClientID[clientID] = clientCids
	}
	// todo: module properties
	// m.UpdateProperty(PropName_NrConnections, len(m.connectionsByClcid))
}

// RemoveConnection removes the connection by its connectionID.
//
// This will close the connnection if it isn't closed already.
// This submits a ServerDisconnectEvent to subscribers.
func (srv *TransportServerBase) RemoveConnection(c IConnection) {
	// protected block
	prot := func() {
		srv.cmux.Lock()
		defer srv.cmux.Unlock()
		srv.removeConnection(c)
	}
	prot()
	// notify listeners
	// publish a notification about the connection
	connectionInfo := ConnectionInfo{
		ClientID:     c.GetClientID(),
		ConnectionID: c.GetConnectionID(),
	}
	senderID := srv.thingID
	thingID := srv.thingID
	notif := msg.NewNotificationMessage(senderID, msg.AffordanceTypeEvent, thingID,
		ServerDisconnectEvent, connectionInfo)
	srv.ForwardNotification(notif)
}

// SendNotification [agent] server sends a notification to its connections
// The connection handles subscriptions.
func (srv *TransportServerBase) SendNotification(notif *msg.NotificationMessage) {
	srv.ForEachConnection(func(c IConnection) {
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
func (srv *TransportServerBase) SendRequest(
	agentID string, req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	c := srv.GetConnectionByClientID(agentID)
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
func (srv *TransportServerBase) SendResponse(
	clientID, cid string, resp *msg.ResponseMessage) (err error) {
	// var c IConnection

	c := srv.GetConnectionByConnectionID(clientID, cid)

	// if nothing to do
	if c == nil {
		err = fmt.Errorf("SendResponse: connection for clientID '%s' and connectionID '%s' not found", clientID, cid)
		return err
	}
	err = c.SendResponse(resp)
	return err
}

// Set the handler that will receive notifications received from the remote agent
func (srv *TransportServerBase) SetNotificationSink(consumer msg.NotificationHandler) {
	srv.notificationSink = consumer
}

// Set the hook to invoke with received requests directed at this module
// Any other requests received by HandleRequest will be forwarded to the sink.
func (m *TransportServerBase) SetAppRequestHook(hook msg.RequestHandler) {
	m.appRequestHook = hook
}

// Set the handler that will receive requests received from the client
func (srv *TransportServerBase) SetRequestSink(sink msg.RequestHandler) {
	// to be determined if there is a use-case for replacing the sink
	if srv.requestSink != nil {
		slog.Warn("SetRequestSink: Overriding existing request sink",
			"module", fmt.Sprintf("%T", srv))
	}
	srv.requestSink = sink
}

// NewTransportServerBase initializes a server module intended for embedding.
//
//	thingID is the transport module instance ID for connect/disconnect notifications
//	subprotocol optional name for including in form operations
//	connectURL is the URL this module can be reached at. Used to set TD.Base
//	authenticator used to include the security in TDs
func NewTransportServerBase(
	thingID, connectURL string, authenticator IAuthenticator) *TransportServerBase {

	base := &TransportServerBase{
		thingID:       thingID,
		authenticator: authenticator,
		connectURL:    connectURL,
		RnrChan:       msg.NewRnRChan(),
	}
	return base
}
