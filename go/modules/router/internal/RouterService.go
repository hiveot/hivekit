package internal

import (
	"crypto/x509"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/router"
	"github.com/hiveot/hivekit/go/modules/transport"
)

type RouterService struct {
	*modules.HiveModuleBase

	// The CA certificate used to verify device connections
	caCert *x509.Certificate

	// handler that provides a TD for the given thingID
	getTD func(thingID string) *td.TD

	// device credentials store
	credStore *CredentialsStore

	// established device connections by origin (schema://host:port)
	cmux              sync.RWMutex
	deviceConnections map[string]transport.ITransportClient

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

// Handle client connection notifications from clients.
// Pass all other notifications upstream.
// TODO: Decide whether this is the right approach on handling client disconnects
//
//		instead of a callback. why not use callbacks?
//		a. allow use of reconnect module instead of handling it here
//		    cant use reconnect as it only supports 1 connection for re-subscribing
//	     no need for reconnect as subscription is always all events/props
// func (m *RouterService) HandleClientNotification(notif *msg.NotificationMessage) {
// 	// todo: should the thingID be used?
// 	if notif.Name == transport.ClientConnectionStatusEvent {
// 		var status transport.ConnectionStatus
// 		var c transport.ITransportClient
// 		var origin string

// 		// need to find the client to update. simple iteration is this is not frequent
// 		m.cmux.Lock()
// 		defer m.cmux.Unlock()
// 		for origin, c = range m.deviceConnections {
// 			if c.GetThingID() == notif.ThingID {
// 				_ = notif.Decode(&status)
// 				_ = origin
// 				go m.HandleClientStatus(status, c)
// 				// client connection status updates are not forwarded as they are owned by the router
// 				return
// 			}
// 		}
// 		// Unexpected this notification is not from a known client
// 		slog.Error("Received connection status event but client thingID is unknown",
// 			"thingID", notif.ThingID)
// 		return
// 	} else {
// 		m.HandleNotification(notif)
// 	}
// }

// HandleRequest handles module requests or routes the request to its destination
func (m *RouterService) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var resp *msg.ResponseMessage

	if req.ThingID != m.GetThingID() {
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
//     the agent's RC connection to the server and forward the request.
//  3. If the TD points to a non-agent device then establish a connection or re-use
//     an existing connection from the pool.
func (m *RouterService) RouteRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// the requested thingID must be known
	tdDoc := m.getTD(req.ThingID)
	if tdDoc == nil {
		// thingID not known, only option is to forward the request downstream
		err = m.ForwardRequest(req, replyTo)
		if err != nil {
			err = fmt.Errorf("RouteRequest: No TD document found for thing '%s' and forwarding failed: %w", req.ThingID, err)
			slog.Warn("RouteRequest", "err", err.Error())
		}
		return err
	}

	// if the tdoc has an agentID then look for its RC connection
	agentID := tdDoc.GetAgentID()
	if agentID != "" {
		c := m.GetRCConnection(agentID)
		if c == nil {
			err = fmt.Errorf("RouteRequest: Unable to connection with agent '%s'", agentID)
		} else {
			err = c.SendRequest(req, replyTo)
		}
	} else {
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

	thingID := router.DefaultRouterThingID
	m := &RouterService{
		HiveModuleBase:    modules.NewHiveModuleBase(thingID, 0),
		caCert:            caCert,
		getTD:             getTD,
		storageDir:        storageDir,
		tpServers:         tpServers,
		deviceConnections: make(map[string]transport.ITransportClient),
		timeout:           timeout,
	}

	var _ router.IRouterService = m // interface check

	return m
}
