package internal

import (
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	reconnect_service "github.com/hiveot/hivekit/go/modules/reconnect/service"
	"github.com/hiveot/hivekit/go/modules/router"
	"github.com/hiveot/hivekit/go/modules/transport/clients"
	"github.com/teris-io/shortid"
)

type RouterServiceImpl struct {
	*modules.HiveModuleBase

	// autoReconnect insert a reconnect client before the transport client
	autoReconnect bool

	// The CA certificate used to verify device connections
	caCert *x509.Certificate

	// handler that provides a TD for the given thingID
	getTD func(thingID string) *td.TD

	// device credentials store
	credStore *CredentialsStore

	// mutex for access to deviceConnections
	cmux sync.RWMutex

	// established device connections by origin (schema://host:port)
	// when connecting to individual things, thingID could be used, however when a device manages
	// multiple Things, they all should use the same connection.
	// The only thing they have in common is the href origin.
	deviceConnections map[string]api.IHiveModule

	// Cache of thingID to origin
	// Used to quickly find the connection of a device.
	thingOrigin map[string]string

	// the preferred protocol to use when creating a new client connection
	preferredProtocol string

	// directory to store device accounts
	storageDir string
	// location of the device credentials store. "" for in-memory only.
	storageFile string

	// handler to get available transport servers for forwarding to RC clients.
	// nil to not support RC.
	getSrv func() []api.ITransportServer
}

// Add the secret to access a Thing.
// if thingID is empty then the credentials are used for all unknown devices.
func (m *RouterServiceImpl) AddDeviceCredential(
	thingID string, clientID string, secret string, secScheme string) {
	m.credStore.AddCredentials(thingID, clientID, secret, secScheme)
}

// Remove the secret to access a Thing
func (m *RouterServiceImpl) DeleteThingCredential(thingID string) {
	m.credStore.DeleteCredentials(thingID)
}

// GetClientConnection returns a client for sending requests to the server with
// the given TD. If a connection doesn't exists then create it.
//
// Previous connections are re-used. This uses schema://host:port (origin) to identify
// the connection. For each TD its origin is cached to avoid repeated lookup of forms.
// Origin lookup can be disabled in case TDs have multiple origins.
//
// If the 'reconnect' option is configured then this returns a Reconnect client
// that automatically reconnects and resubscribes if the connection fails.
//
// The caller must check if the connection is established before sending a message.
//
//	tdoc is the TD of the device to connect to
//	op is the operation to perform
//	name is the optional affordance name for the operation. "" for thing level operations.
func (m *RouterServiceImpl) GetClientConnection(
	tdoc *td.TD, op string, name string) (cl api.IHiveModule, err error) {

	var c api.ITransportClient
	var form *td.Form
	var href string
	var match bool

	// 1. locate the existing connection using TD's origin
	// this lock should really be per device
	m.cmux.Lock()
	thingOrigin, found := m.thingOrigin[tdoc.ID]
	if found {
		cl, found = m.deviceConnections[thingOrigin]
	}
	defer m.cmux.Unlock()

	// 2. If a valid connection was not found. Redetermine origin and reconnect
	if !found {
		prefScheme := api.WotWebsocketScheme
		prefSubprotocol := api.WotWebsocketSubprotocol
		protocolParts := strings.Split(m.preferredProtocol, ":")
		if len(protocolParts) > 1 {
			prefScheme = protocolParts[0]
			prefSubprotocol = protocolParts[1]
		}
		//
		if op == "" {
			form, match = tdoc.GetConnectForm(prefScheme, prefSubprotocol)
		} else {
			form, match = tdoc.GetForm(op, name, prefScheme, prefSubprotocol)
		}
		_ = match
		if form == nil {
			return nil, fmt.Errorf("No matching form for connecting to Thing '%s'", tdoc.ID)
		}
		// get the full URL for the operation
		href, err = form.GetHRef(tdoc.Base, nil)

		// if an href cannot be determined then this can't continue
		if err != nil {
			return nil, fmt.Errorf("No href for operation '%s' in TD '%s'", op, tdoc.ID)
		}
		// determine the origin that identifies the client connection
		urlParts, err := url.Parse(href)
		if err != nil {
			return nil, err
		}
		newOrigin := fmt.Sprintf("%s://%s", urlParts.Scheme, urlParts.Host)
		// origin on UDS (unix) must include the path while on tcp/http/mqtt it is schema://host:port
		if strings.ToLower(urlParts.Scheme) == "unix" {
			// Warning, a unix socket connection is local to the machine only.
			newOrigin = href
		}
		// store the Thing's origin for future quick lookup
		m.thingOrigin[tdoc.ID] = newOrigin
		if newOrigin != thingOrigin {
			cl, found = m.deviceConnections[newOrigin]
			// guard against orphan connection if the origin changed after a TD update and it already has a connection
			if found {
				slog.Error("GetClientConnection: found existing connection after unexpected change in TD origin. Closing old connection.")
				cl.Stop()
			}
		}

		// 3. Create a new client for the origin
		// TODO: how to determine the CA and client cert for this server?
		c, err = clients.NewTransportClientFromForm(tdoc, form, m.caCert)
		if err != nil {
			return nil, err
		}
		c.SetTimeout(m.GetTimeout())
		// Authentication fails if no credentials can be matched
		err = c.AuthenticateWithForm(tdoc, m.credStore.GetCredentials)
		if err != nil {
			// No auth. Discard this connection.
			slog.Warn("GetClientConnection: authentication failed", "ThingID", tdoc.ID, "err", err.Error())
			return nil, err
		}
		if m.autoReconnect {
			// reconnect connects the client on start
			rc := reconnect_service.NewReconnectService(c)
			cl = rc
		} else {
			cl = c
		}
		m.deviceConnections[newOrigin] = cl

		// forward notifications to this module and up to its consumer
		cl.SetNotificationSink(m)
		// last, Connect. If reconnect is used it will connect the client
		err = cl.Start()
	}

	return cl, err
}

// Return the reverse-client connection to a device, if it exists.
// This returns nil if the clientID does not have an existing connection.
func (m *RouterServiceImpl) GetRCConnection(clientID string) (c api.IConnection) {
	if m.getSrv == nil {
		return nil
	}
	serverList := m.getSrv()
	for _, tp := range serverList {
		c := tp.GetConnectionByClientID(clientID)
		if c != nil {
			return c
		}
	}
	return nil
}

// HandleRequest handles module requests or routes the request to its destination
func (m *RouterServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
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
func (m *RouterServiceImpl) HasThingCredentials(thingID string) bool {
	return m.credStore.HasCredentials(thingID)
}

// Determine if the thing is reachable by the router.
//
// This returns true if a client connection is established by the router, or if
// a reverse connection exists by the thing deviceID.
// func (m *RouterServiceImpl) IsReachable(thingID string) bool {
// 	return false
// }

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
//  2. If the TD contains an RC clientID, injected by the directory module, then lookup
//     the device's RC connection to the server and forward the request.
//  3. If the TD points to a non RC device then establish a connection or re-use
//     an existing connection from the pool.
func (m *RouterServiceImpl) RouteRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// the requested thingID must be known
	tdoc := m.getTD(req.ThingID)
	if tdoc == nil {
		// thingID not known, only option is to forward the request downstream
		err = m.ForwardRequest(req, replyTo)
		if err != nil {
			err = fmt.Errorf("RouteRequest: No TD document found for thing '%s' and forwarding failed: %w", req.ThingID, err)
			// just log as info as this can be legit.
			slog.Info("RouteRequest", "err", err.Error())
		}
		return err
	}

	// if the tdoc has an RC clientID then look for its RC connection
	rcClientID := tdoc.GetRCClientID()
	if rcClientID != "" {
		c := m.GetRCConnection(rcClientID)
		if c == nil {
			err = fmt.Errorf("RouteRequest: device '%s' isnt connected", rcClientID)
		} else {
			err = c.SendRequest(req, replyTo)
		}
	} else {
		c, err2 := m.GetClientConnection(tdoc, req.Operation, req.Name)
		// TODO: tdoc/op/name provides an form with href, but this isnt used in
		// HandleRequest. Is this a problem?
		// Depends on the protocol?
		// 1. should c handlerequest have a callback to get href?
		// 2. should handlerequest use a context to pass href?
		if c == nil {
			slog.Warn("RouteRequest: Unable to establish a connection to client", "err", err2)
			err = err2
		} else if err2 != nil {
			slog.Warn("RouteRequest: Connection failed", "err", err2)
			err = err2
		} else {
			err = c.HandleRequest(req, replyTo)
		}
	}

	return err
}

// Start the router module.
// This loads to stored Thing credentials
func (m *RouterServiceImpl) Start() (err error) {
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
func (m *RouterServiceImpl) Stop() {
	slog.Info("Stop: Stopping router module")
	for clientID, c := range m.deviceConnections {
		_ = clientID
		c.Stop()
	}
	m.deviceConnections = nil
	// last close credential store
	m.credStore.Close()
}

// NewRouterServiceImpl creates a new router module
//
// Use getSrv if routing requests to server RC connected device should be supported.
// AutoReconnect will attempt to automatically reconnect failed client connections. Note that this
// might hide authentication problems.
//
//	storageDir with the module credentials storage directory, "" for in-memory testing
//	autoReconnect flag, to enable auto-reconnect on client connections
//	getTD  handler to lookup a TD for a thingID from a directory
//	getSrv handler returning a list of transport servers that can contain RC devices.
//	caCert is the CA used to verify device connections
//	timeout is the maximum communication timeout with connect clients
func NewRouterServiceImpl(
	storageDir string,
	autoReconnect bool,
	getTD func(thingID string) *td.TD,
	getSrv func() []api.ITransportServer,
	caCert *x509.Certificate, timeout time.Duration,
) *RouterServiceImpl {
	if timeout == 0 {
		timeout = msg.DefaultRnRTimeout
	}

	// defaultClientID := "router"
	thingID := router.DefaultRouterThingID + "-" + shortid.MustGenerate()
	m := &RouterServiceImpl{
		HiveModuleBase:    modules.NewHiveModuleBase(thingID, timeout),
		autoReconnect:     autoReconnect,
		caCert:            caCert,
		getTD:             getTD,
		preferredProtocol: api.WotMqttProtocolType, //WotWebsocketProtocolType,
		storageDir:        storageDir,
		getSrv:            getSrv,
		deviceConnections: make(map[string]api.IHiveModule),
		thingOrigin:       make(map[string]string),
	}

	var _ router.IRouterService = m // interface check

	return m
}
