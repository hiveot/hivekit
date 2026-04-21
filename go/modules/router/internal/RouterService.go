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
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/modules/router"
	"github.com/hiveot/hivekit/go/modules/transports"
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
	deviceConnections map[string]transports.ITransportClient

	// the thingID of this service for messaging
	routerThingID string

	// directory to store device accounts
	storageDir string
	// location of the device credentials store. "" for in-memory only.
	storageFile string

	// client communication timeout or 0 for default
	timeout time.Duration

	// transport servers
	tpServers []transports.ITransportServer
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
	c transports.ITransportClient, err error) {

	// use URI scheme to determine the protocol, except for the hiveot WSS, which also
	// has a wss scheme. Instead look at the base path which is fixed.
	protocolType, href := clients.GetProtocolType(tdi)
	parts, err := url.Parse(href)
	if err != nil {
		return nil, err
	}
	connID := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	c, found := m.deviceConnections[connID]
	if !found || !c.IsConnected() {
		// TODO: how to determine the CA for this server?
		// TODO: support use of client cert for this server?
		// connect and store the connection if successful
		c, err = clients.NewTransportClient(protocolType, href, m.caCert, nil)
		c.SetTimeout(m.timeout)
		if err == nil {
			err = c.Authenticate(tdi, m.credStore.GetCredentials)
		}
		if err == nil {
			m.deviceConnections[connID] = c
			// forward notifications from devices to this module and up to its consumer
			c.SetNotificationSink(m.HandleNotification)
		} else {
			c = nil // auth failed
			err = fmt.Errorf("Router:GetClientConnection. Connection failed: %w", err)
		}
	}
	return c, err
}

// Return the reverse-client connection to an agent, if it exists.
// This returns nil if the clientID does not have an existing connection.
func (m *RouterService) GetRCConnection(clientID string) (c transports.IConnection) {
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
		err := fmt.Errorf("Unhandled request: thingID='%s', op='%s', name='%s", req.ThingID, req.Operation, req.Name)
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
	tdi := m.getTD(req.ThingID)
	if tdi == nil {
		return m.ForwardRequest(req, replyTo)
	}
	// c := clients.ConnectToThing(tdi, m.getCredentials)

	agentID := tdi.GetAgentID()
	forms := tdi.GetForms(req.Operation, req.Name)
	if len(forms) > 0 {
		// TBD right now just use the first form.
		href, err = tdi.GetFormHRef(forms[0], nil)
	}
	// if no form was found then simply use the Base attribute
	if href == "" {
		href = tdi.Base
	}
	// without href attempt looking up a reverse connection
	if href == "" && agentID == "" {
		err = fmt.Errorf("No connection information in TD for Thing '%s'", req.ThingID)
	} else if href == "" {
		c := m.GetRCConnection(agentID)
		if c == nil {
			err = fmt.Errorf("Unable to connection with agent '%s'", agentID)
		} else {
			err = c.SendRequest(req, replyTo)
		}
	} else {
		c, err2 := m.GetClientConnection(tdi)
		if c == nil {
			err = fmt.Errorf("Unable to establish a connection to client '%s': %w", agentID, err2)
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
func NewRouterService(storageDir string,
	getTD func(thingID string) *td.TD,
	tpServers []transports.ITransportServer,
	caCert *x509.Certificate,
) *RouterService {

	m := &RouterService{
		caCert:            caCert,
		getTD:             getTD,
		storageDir:        storageDir,
		tpServers:         tpServers,
		deviceConnections: make(map[string]transports.ITransportClient),
		routerThingID:     router.DefaultRouterThingID,
		timeout:           msg.DefaultRnRTimeout,
	}

	var _ router.IRouterService = m // interface check

	return m
}
