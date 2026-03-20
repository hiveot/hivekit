package internal

import (
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"sync"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/clients"
	routerapi "github.com/hiveot/hivekit/go/modules/router/api"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot/td"
)

// Login credentials for known devices
type DeviceAccount struct {
	ClientID string `json:"clientID"`

	// Secre password or token
	Secret string `json:"secret"`
}

type RouterModule struct {
	modules.HiveModuleBase

	// The CA certificate used to verify device connections
	caCert *x509.Certificate

	// handler that provides a TD for the given thingID
	getTD func(thingID string) *td.TD

	// transport servers
	tpServers []transports.ITransportServer

	// deviceAccounts holds the credentials of devices by thingID
	mux            sync.RWMutex
	deviceAccounts map[string]DeviceAccount

	// location of the encrypted device account store
	storageRoot       string
	deviceStorageFile string

	// established device connections
	deviceConnections map[string]transports.IClientConnection
}

// Add the secret to access a Thing.
func (m *RouterModule) AddThingCredential(
	thingID string, clientID string, secret string) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.deviceAccounts[thingID] = DeviceAccount{ClientID: clientID, Secret: secret}
}

// Remove the secret to access a Thing
func (m *RouterModule) DeleteThingCredential(thingID string) {
	m.mux.Lock()
	defer m.mux.Unlock()
	delete(m.deviceAccounts, thingID)
}

// Obtain the connection credentials for connection to the device
func (m *RouterModule) GetDeviceCredentials(destination *td.TD) (
	clientID string, token string, err error) {
	thingID := destination.ID
	acct, found := m.deviceAccounts[thingID]
	if !found {
		return "", "", fmt.Errorf("Unknown thing with ID '%s'", destination.ID)
	}
	return acct.ClientID, acct.Secret, nil
}

// Return a client connection to the given href.
//
// This uses schema://host:port to identify the connection to use.
// If a connection to this client already exists then use it, otherwise create it.
//
// This returns an error if no connection can be established.
func (m *RouterModule) GetClientConnection(tdi *td.TD) (
	c transports.IClientConnection, err error) {

	href := tdi.Base
	parts, err := url.Parse(href)
	if err != nil {
		return nil, err
	}
	connID := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	c, found := m.deviceConnections[connID]
	if !found || !c.IsConnected() {
		// fixme: how to determine the CA for this server?
		// connect and store the connection if successful
		c, err = clients.NewTransportClient(href, m.caCert, nil)
		if err == nil {
			err = c.Authenticate(tdi, m.GetDeviceCredentials)
		}
		if err == nil {
			m.deviceConnections[connID] = c
		} else {
			c = nil // auth failed
			slog.Error("Router:GetClientConnection. Connection failed", "err", err.Error())
		}
	}
	return c, err
}

// Return the reverse-client connection to an agent, if it exists.
// This returns nil if the clientID does not have an existing connection.
func (m *RouterModule) GetRCConnection(clientID string) (c transports.IConnection) {
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
func (m *RouterModule) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var resp *msg.ResponseMessage

	if req.ThingID != m.GetModuleID() {
		return m.RouteRequest(req, replyTo)
	}
	// handle requests for router module itself
	switch req.Operation {

	// case wot.OpReadProperty:
	// 	resp, err = m.ReadProperty(req)
	// case wot.OpReadMultipleProperties:
	// 	resp, err = m.ReadMultipleProperties(req)
	// case wot.OpReadAllProperties:
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

// Return a flag indicating whether the credentials are set for a Thing
func (m *RouterModule) HasThingCredential(thingID string) bool {
	m.mux.RLock()
	defer m.mux.RUnlock()
	_, found := m.deviceAccounts[thingID]
	return found
}

// Determine if the thing is reachable by the router.
//
// This returns true if a client connection is established by the router, or if
// a reverse connection exists by the thing agent.
func (m *RouterModule) IsReachable(thingID string) bool {
	return false
}

// Return the ISO timestamp when the Thing was last seen by the router.
// This returns an empty string if no known record exists.
func (m *RouterModule) LastSeen(thingID string) string {
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
func (m *RouterModule) RouteRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
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
	if href == "" {
		href = tdi.Base
	}
	// without href attempt looking up a reverse connection
	if href == "" {
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

// Start the router module.
// This loads to stored data
func (m *RouterModule) Start(_ string) (err error) {
	m.deviceStorageFile = filepath.Join(m.storageRoot, m.GetModuleID())
	m.deviceAccounts = make(map[string]DeviceAccount)
	// TODO: load keys.
	// k, err := keyloader.LoadKey("storage")

	return err
}

// Stop the router module.
// This closes all established client connections.
func (m *RouterModule) Stop() {
	for clientID, c := range m.deviceConnections {
		_ = clientID
		c.Close()
	}
	// save keys
	m.deviceConnections = nil
}

// NewRouterModule creates a new router module
//
//	storageRoot is the root directory where modules create their storage, "" for in-memory testing
//	getTD is the handler to lookup a TD for a thingID from a directory
//	transports is a list of transport servers that can contain reverse agent connections.
//	caCert is the CA used to verify device connections
func NewRouterModule(storageRoot string,
	getTD func(thingID string) *td.TD,
	tpServers []transports.ITransportServer,
	caCert *x509.Certificate) *RouterModule {

	m := &RouterModule{
		caCert:    caCert,
		getTD:     getTD,
		tpServers: tpServers,
	}
	m.SetModuleID(routerapi.DefaultRouterServiceID)

	var _ routerapi.IRouterModule = m // interface check

	return m
}
