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
	reconnectpkg "github.com/hiveot/hivekit/go/modules/reconnect/pkg"
	"github.com/hiveot/hivekit/go/modules/router"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/clients"
)

type RouterService struct {
	*modules.HiveModuleBase

	// autoReconnect insert a reconnect client before the transport client
	autoReconnect bool

	// The CA certificate used to verify device connections
	caCert *x509.Certificate

	// handler that provides a TD for the given thingID
	getTD func(thingID string) *td.TD

	// device credentials store
	credStore *CredentialsStore

	// established device connections by origin (schema://host:port)
	cmux sync.RWMutex
	// deviceConnections map[string]transport.ITransportClient
	deviceConnections map[string]modules.IHiveModule

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
func (m *RouterService) AddDeviceCredential(
	thingID string, clientID string, secret string, secScheme string) {
	m.credStore.AddCredentials(thingID, clientID, secret, secScheme)
}

// Remove the secret to access a Thing
func (m *RouterService) DeleteThingCredential(thingID string) {
	m.credStore.DeleteCredentials(thingID)
}

// GetClientConnection returns a module for sending requests to the server with
// the given TD. If a connection doesn't exists then create it.
//
// This uses schema://host:port (origin) to identify the connection to use.
// If a connection to this client already exists then use it, otherwise create it.
//
// If the 'reconnect' option is configured then this returns a Reconnect client
// that automatically reconnects and resubscribes if the connection fails.
//
// The caller must check if the connection is established before sending a message.
func (m *RouterService) GetClientConnection(tdi *td.TD, op string) (cl modules.IHiveModule, err error) {

	var c transport.ITransportClient

	// use URI scheme to determine the protocol, except for the hiveot WSS, which also
	// has a wss scheme. Instead look at the base path which is fixed.
	protocolType, href := clients.GetProtocolType(tdi, op)
	parts, err := url.Parse(href)
	if err != nil {
		return nil, err
	}
	// determine the 'origin' for this connection, which is the protocol and address
	// of the connection. Multiple Things from the same agent share the same connection.
	origin := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	m.cmux.Lock()
	cl, found := m.deviceConnections[origin]
	if !found {
		// TODO: how to determine the CA for this server?
		// TODO: support use of client cert for this server?
		c, err = clients.NewTransportClient(protocolType, href, m.caCert)
		if err != nil {
			return nil, err
		}
		c.SetTimeout(m.timeout)
		err = c.AuthenticateWithForm(tdi, m.credStore.GetCredentials)
		if err != nil {
			return nil, err
		}
		if m.autoReconnect {
			rc := reconnectpkg.NewReconnectClient(c)
			cl = rc
		} else {
			cl = c
		}
		m.deviceConnections[origin] = cl
		// forward notifications to this module and up to its consumer
		cl.SetNotificationSink(m)
		err = cl.Start()
	}
	m.cmux.Unlock()

	// if rc.GetConnectionStatus() != transport.StatusConnected {
	// 	err = rc.AuthenticateWithForm(tdi, m.credStore.GetCredentials)
	// 	if err == nil {
	// 		slog.Info("GetClientConnection: (re)Connecting to ", slog.String("href", href))
	// 		err = rc.Connect()
	// 	}
	// 	if err != nil {
	// 		err = fmt.Errorf("GetClientConnection. Connection to '%s' failed: %w", origin, err)
	// 		slog.Warn(err.Error())
	// 	}
	// }
	return cl, err
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

	if req.ThingID != m.GetThingID() {
		return m.RouteRequest(req, replyTo)
	}
	// handle requests for router module itself
	switch req.Operation {
	// nothing supported yet, add some properties on nr clients
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
// func (m *RouterService) LastSeen(thingID string) string {
// 	return ""
// }

// // Route the request to its destination:
//
// Lookup the TD of the ThingID and determine its destination:
//
//  1. If no TD exists then simply forward the request to the request sink
//  2. If the TD contains an agentID, injected by the digitwin module, then lookup
//     the agent's RC connection to the server and forward the request.
//  3. If the TD points to a non-agent device then establish a connection or re-use
//     an existing connection from the pool.
func (m *RouterService) RouteRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// the requested thingID must be known
	tdoc := m.getTD(req.ThingID)
	if tdoc == nil {
		// thingID not known, only option is to forward the request downstream
		err = m.ForwardRequest(req, replyTo)
		if err != nil {
			err = fmt.Errorf("RouteRequest: No TD document found for thing '%s' and forwarding failed: %w", req.ThingID, err)
			slog.Warn("RouteRequest", "err", err.Error())
		}
		return err
	}

	// if the tdoc has an agentID then look for its RC connection
	agentID := tdoc.GetAgentID()
	if agentID != "" {
		c := m.GetRCConnection(agentID)
		if c == nil {
			err = fmt.Errorf("RouteRequest: agent '%s' isnt connected", agentID)
		} else {
			err = c.SendRequest(req, replyTo)
		}
	} else {
		c, err2 := m.GetClientConnection(tdoc, req.Operation)
		if c == nil {
			err = fmt.Errorf("RouteRequest: Unable to establish a connection to client '%s': %w", agentID, err2)
		} else {
			err = c.HandleRequest(req, replyTo)
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
		c.Stop()
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

	thingID := router.DefaultRouterThingID
	m := &RouterService{
		HiveModuleBase:    modules.NewHiveModuleBase(thingID, 0),
		autoReconnect:     true,
		caCert:            caCert,
		getTD:             getTD,
		storageDir:        storageDir,
		tpServers:         tpServers,
		deviceConnections: make(map[string]modules.IHiveModule),
		timeout:           timeout,
	}

	var _ router.IRouterService = m // interface check

	return m
}
