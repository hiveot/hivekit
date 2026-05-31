package internal

import (
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/router"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/clients"
)

type RouterService struct {
	modules.HiveModuleBase

	// The CA certificate used to verify device connections
	caCert *x509.Certificate

	// handler that provides a TD for the given thingID
	getTD func(thingID string) *td.TD

	// device credentials store
	credStore *CredentialsStore

	// established device connections
	cmux              sync.RWMutex
	deviceConnections map[string]transport.ITransportClient

	// the thingID of this service for messaging
	routerThingID string

	// directory to store device accounts
	storageDir string
	// location of the device credentials store. "" for in-memory only.
	storageFile string

	// client communication timeout or 0 for default
	timeout time.Duration

	// transport servers
	tpServers []transport.ITransportServer
}

// Add the secret to access a Thing.
func (m *RouterService) AddThingCredential(
	thingID string, clientID string, secret string, secScheme string) {
	m.credStore.AddCredentials(thingID, clientID, secret, secScheme)
}

// Remove the secret to access a Thing
func (m *RouterService) DeleteThingCredential(thingID string) {
	m.credStore.DeleteCredentials(thingID)
}

// Return a client connection to the given href.
//
// This uses schema://host:port to identify the connection to use.
// If a connection to this client already exists then use it, otherwise create it.
//
// This returns an error if no connection can be established.
func (m *RouterService) GetClientConnection(tdi *td.TD) (
	c transport.ITransportClient, err error) {

	// use URI scheme to determine the protocol, except for the hiveot WSS, which also
	// has a wss scheme. Instead look at the base path which is fixed.
	protocolType, href := clients.GetProtocolType(tdi)
	parts, err := url.Parse(href)
	if err != nil {
		return nil, err
	}
	connID := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	c, found := m.deviceConnections[connID]
	if !found || c.GetConnectionStatus() != transport.StatusConnected {
		// TODO: how to determine the CA for this server?
		// TODO: support use of client cert for this server?
		// connect and store the connection if successful
		c, err = clients.NewTransportClient(protocolType, href, m.caCert)
		c.SetTimeout(m.timeout)

		// TODO: do clients emit notifications? yes connect/disconnect
		// do clients accept requests? yes connect action, status props
		// how are clients modules identified (thingID)? client type, uuid, both, clientID, cid?
		//  not clientID as the same ID can be used to connect to different devices
		//  not client type as these are the same for all instances
		//  uuid is possible but is harder in testing/debugging
		//  both appended could work
		//  cid (shortid) is used in identifying the client instance to the server (clientid-cid)
		//     careful for hidden dependencies
		//     can the same TLS client (cid) be used by multiple transports like on the server?
		//       eg use same TLS client for SSE and WSS connection?
		//         in theory yes, in practise a new instance is more l
		//  {protocolType}-{clientID}-{shortid}
		//
		// how to reconnect? use the thingID from the notification in the connect request
		// can reconnect be placed before the router? yes
		//
		// does the router remove connections when they drop?  no, only on close.
		//
		// should router reconnect?
		//  A: yes, it knows what need to be done and manages subscriptions
		//  B: no, use the reconnect module which resubscribes
		//
		// does each client have a unique thingID?
		//  a. no, thingID not used/needed?
		//  b. yes, thingID used in notifications
		//      needed because router can connect using multiple clients
		//
		// can connections be managed remotely? use-case?
		//
		// should client modules deal with connect notifications?
		//  a. yes, allows reconnect module to work
		//     yes, supports retrieving stats on reconnection attempts
		//     yes, making clients addressable allows for remote configuration
		//  b. no, bit of a hassle, could implement reconnect with resubscribe itself
		//
		if err == nil {
			err = c.AuthenticateWithForm(tdi, m.credStore.GetCredentials)
		}
		if err == nil {
			err = c.Connect()
		}
		if err == nil {
			m.deviceConnections[connID] = c
			// forward notifications from devices to this module and up to its consumer
			c.SetNotificationSink(m.HandleNotification)
		} else {
			c = nil // auth failed
			err = fmt.Errorf("RouterService.GetClientConnection. Connection failed: %w", err)
		}
	}
	return c, err
}

// Return the reverse-client connection to an agent, if it exists.
// This returns nil if the clientID does not have an existing connection.
func (m *RouterService) GetRCConnection(clientID string) (c transport.IConnection) {
	if m.tpServers == nil {
		return nil
	}
	for _, tp := range m.tpServers {
		c := tp.GetConnectionByClientID(clientID)
		if c != nil {
			return c
		}
	}
	return nil
}

// HandleRequest handles module requests or routes the request to its destination
func (m *RouterService) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var resp *msg.ResponseMessage

	if req.ThingID != m.routerThingID {
		return m.RouteRequest(req, replyTo)
	}
	// handle requests for router module itself
	switch req.Operation {

	// case td.OpReadProperty:
	// 	resp, err = m.ReadProperty(req)
	// case td.OpReadMultipleProperties:
	// 	resp, err = m.ReadMultipleProperties(req)
	// case td.OpReadAllProperties:
	// 	resp, err = m.ReadAllProperties(req)
	// directory specific operations could be handled here
	default:
		err := fmt.Errorf("RouterService.HandleRequest: Unhandled request: thingID='%s', op='%s', name='%s", req.ThingID, req.Operation, req.Name)
		slog.Warn(err.Error())
	}
	if resp != nil {
		err = replyTo(resp)
	}
	return err
}

// HasDeviceCredentials returns a flag if credentials are set for a Thing
func (m *RouterService) HasThingCredentials(thingID string) bool {
	return m.credStore.HasCredentials(thingID)
}

// Determine if the thing is reachable by the router.
//
// This returns true if a client connection is established by the router, or if
// a reverse connection exists by the thing agent.
func (m *RouterService) IsReachable(thingID string) bool {
	return false
}

// Return the ISO timestamp when the Thing was last seen by the router.
// This returns an empty string if no known record exists.
func (m *RouterService) LastSeen(thingID string) string {
	return ""
}

// Route the request to its destination:
//
// Lookup the TD of the ThingID and determine its destination:
//
//  1. If no TD exists then simply forward the request to the request sink
//  2. If the TD contains an agentID, injected by the digitwin module, then lookup
//     the agent's connection to the server and forward the request.
//  3. If the TD points to a non-agent device then establish a connection or re-use
//     an existing connection from the pool.
func (m *RouterService) RouteRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var href string
	var f *td.Form

	tdDoc := m.getTD(req.ThingID)
	if tdDoc == nil {
		err = m.ForwardRequest(req, replyTo)
		if err != nil {
			err = fmt.Errorf("RouteRequest: No TD document found for thing '%s' and forwarding failed: %w", req.ThingID, err)
			slog.Warn("RouteRequest", "err", err.Error())
		}
		return err
	}
	// c := clients.ConnectToThing(tdi, m.getCredentials)

	agentID := tdDoc.GetAgentID()
	// determine the first supported protocol that matches
	for _, clientProtocol := range clients.SupportedClientProtocols {
		f, href, err = tdDoc.GetFormHRef(req.Operation, req.Name, clientProtocol, nil)
		if href != "" {
			break
		}
		_ = f
	}

	// if no form was found then simply use the Base attribute
	if href == "" {
		href = tdDoc.Base
	}
	// without href attempt looking up a reverse connection
	if href == "" && agentID == "" {
		err = fmt.Errorf("RouteRequest: No connection information in TD for Thing '%s'", req.ThingID)
	} else if href == "" {
		c := m.GetRCConnection(agentID)
		if c == nil {
			err = fmt.Errorf("RouteRequest: Unable to connection with agent '%s'", agentID)
		} else {
			err = c.SendRequest(req, replyTo)
		}
	} else {
		// FIXME: use form or protocol ? href?
		c, err2 := m.GetClientConnection(tdDoc)
		if c == nil {
			err = fmt.Errorf("RouteRequest: Unable to establish a connection to client '%s': %w", agentID, err2)
		} else {
			err = c.SendRequest(req, replyTo)
		}
	}
	return err
}

// SetTimeout changes the default communication timeout applied to new connections
// Existing connections are not changed.
func (m *RouterService) SetTimeout(rpcTimeout time.Duration) {
	m.timeout = rpcTimeout
}

// Start the router module.
// This loads to stored Thing credentials
func (m *RouterService) Start() (err error) {
	slog.Info("Start: Starting router module")
	if m.storageDir != "" {
		fileName := "deviceCredentials.json"
		m.storageFile = filepath.Join(m.storageDir, fileName)
	}
	m.credStore = NewCredentialsStore(m.storageFile)
	err = m.credStore.Open()
	return err
}

// Stop the router module.
// This closes all established client connections.
func (m *RouterService) Stop() {
	slog.Info("Stop: Stopping router module")
	for clientID, c := range m.deviceConnections {
		_ = clientID
		c.Close()
	}
	m.deviceConnections = nil
	// last close credential store
	m.credStore.Close()
}

// NewRouterService creates a new router module
//
//	storageDir with the module credentials storage directory, "" for in-memory testing
//	getTD is the handler to lookup a TD for a thingID from a directory
//	transports is a list of transport servers that can contain reverse agent connections.
//	caCert is the CA used to verify device connections
//	timeout is the maximum communication timeout with connect clients
func NewRouterService(storageDir string, getTD func(thingID string) *td.TD,
	tpServers []transport.ITransportServer, caCert *x509.Certificate, timeout time.Duration,
) *RouterService {
	if timeout == 0 {
		timeout = msg.DefaultRnRTimeout
	}

	m := &RouterService{
		caCert:            caCert,
		getTD:             getTD,
		storageDir:        storageDir,
		tpServers:         tpServers,
		deviceConnections: make(map[string]transport.ITransportClient),
		routerThingID:     router.DefaultRouterThingID,
		timeout:           timeout,
	}

	var _ router.IRouterService = m // interface check

	return m
}
