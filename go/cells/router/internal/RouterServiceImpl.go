package internal

import (
	"crypto/tls"
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
	"github.com/hiveot/hivekit/go/cells"
	reconnect_service "github.com/hiveot/hivekit/go/cells/reconnect/service"
	"github.com/hiveot/hivekit/go/cells/router"
	"github.com/hiveot/hivekit/go/cells/transport/clients"
	"github.com/teris-io/shortid"
)

// Implementation of the router service
type RouterServiceImpl struct {
	*cells.HiveCellBase

	// autoReconnect insert a reconnect client before the transport client
	autoReconnect bool

	// default ClientID if no credentials are set
	clientID string

	// The client certificate this service can use to connect to stand-alone devices.
	// NOTE: This has the limitation that these devices must recognize the CA that signed
	//  the certificate.
	clientCert *tls.Certificate

	// mutex for access to deviceConnections
	cmux sync.RWMutex

	// Cache of thingID to connect URL
	// Used to quickly find the connection of a device.
	connectURLByThingID map[string]string

	// connection credentials store
	credStore *CredentialsStore

	// established device connections by origin (schema://host:port)
	// when connecting to individual things, thingID could be used, however when a device manages
	// multiple Things, they all should use the same connection.
	// The only thing they have in common is the href origin.
	deviceConnections map[string]api.IHiveCell

	// handler that provides a TD for the given thingID.
	// Required for setting credentials and connecting to clients.
	getTD func(thingID string) *td.TD

	// handler to get available transport servers for forwarding to RC clients.
	// nil to not support RC.
	getSrv func() []api.ITransportServer

	// the preferred protocol to use when creating a new client connection
	preferredProtocol string

	// The root CA certificates used to verify device connections
	rootCAs *x509.CertPool

	// location of the device credentials store. "" for in-memory only.
	storageFile string
}

// Add the secret to access a Thing.
//
// if thingID is empty then the credentials are used for all devices for which
// no credentials are set. Use with care as it exposes the token to these devices.
func (svc *RouterServiceImpl) AddCredentials(
	thingID string, clientID string, secret string, credType string) {

	creds := ThingCredentials{
		ClientID: clientID,
		Secret:   secret,
		CredType: credType,
	}

	// Set as default credentials if no thingID is provided.
	if thingID == "" {
		svc.credStore.AddCredentials("", creds)
		return
	}

	// determine the connectURL for this thing
	tdoc := svc.getTD(thingID)
	if tdoc == nil {
		slog.Error("AddDeviceCredential: Cant add credentials. TD for thingID not found", "thingID", thingID)
	} else {
		connectURL, _, err := svc.GetConnectURL(tdoc, "", "")
		if err == nil {
			svc.credStore.AddCredentials(connectURL, creds)
		}
	}
	// also store the credentials by thingID .. might need it to recover later
	// svc.credStore.AddCredentials(thingID, creds)
}

// Remove the secret to access a Thing
func (svc *RouterServiceImpl) DeleteCredentials(thingID string) {
	// determine the connectURL for this thing
	tdoc := svc.getTD(thingID)
	if tdoc == nil {
		slog.Warn("DeleteCredentials: Cant delete credentials. TD for thingID not found", "thingID", thingID)
		return
	}
	connectURL, _, err := svc.GetConnectURL(tdoc, "", "")
	if err != nil {
		slog.Warn("DeleteCredentials: TD has no connection information", "thingID", thingID)
		return
	}
	svc.credStore.DeleteCredentials(connectURL)
	// svc.credStore.DeleteCredentials(thingID)
}

// Determine the connection URL from the Thing TD and operation
// In order of preference: websocket first
//
//	tdoc is the TD to get the URL from
//	op is the operation to use
//	name is the optional affordance name
//
// This returns the connect URL, the form used or an error
func (svc *RouterServiceImpl) GetConnectURL(
	tdoc *td.TD, op string, name string) (connectURL string, form *td.Form, err error) {
	var hrefURL *url.URL
	var match bool

	prefScheme := api.WotWebsocketScheme
	prefSubprotocol := api.WotWebsocketSubprotocol

	protocolParts := strings.Split(svc.preferredProtocol, ":")
	if len(protocolParts) > 1 {
		prefScheme = protocolParts[0]
		prefSubprotocol = protocolParts[1]
	}
	// Attempt to find a form matching the preferred protocol
	form, match = tdoc.GetForm(op, name, prefScheme, prefSubprotocol)

	_ = match
	if form == nil {
		return "", nil, fmt.Errorf("GetConnectURL: No matching form for connecting to Thing '%s'", tdoc.ID)
	}
	// get the full URL for the operation
	hrefURL, err = form.ResolveHRef(tdoc.Base, nil)

	// if an href cannot be determined then this can't continue
	if err != nil {
		return "", nil, fmt.Errorf("GetConnectURL: No href for operation '%s' in TD '%s'", op, tdoc.ID)
	}
	// determine the connectURL that identifies the client connection
	urlScheme := strings.ToLower(hrefURL.Scheme)
	if urlScheme == "https" {
		// ignore the path as it differs per operation/name but can use the same client
		connectURL = fmt.Sprintf("%s://%s", hrefURL.Scheme, hrefURL.Host)
	} else {
		// use the full URL as the connection endpoint
		connectURL = hrefURL.String()
	}
	return connectURL, form, nil
}

// GetClientConnection returns a client for sending requests to the server with
// the given TD. If a connection doesn't exists then create it.
//
// Previous connections are re-used. This uses the connect URL to identify
// the connection. For each requested thingID, the connectURL is calculated and
// cached.
//
// Caching of connectURL can be disabled in case TDs are used with different
// connect URLs.
//
// If the 'reconnect' option is configured then this returns a Reconnect client
// that automatically reconnects and resubscribes if the connection fails.
//
// The caller must check if the connection is established before sending a message.
//
//	tdoc is the TD of the device to connect to.
//	op is the operation to perform
//	name is the optional affordance name for the operation. "" for thing level operations.
func (svc *RouterServiceImpl) GetClientConnection(
	tdoc *td.TD, op string, name string) (cl api.IHiveCell, err error) {

	var c api.ITransportClient
	var form *td.Form

	// 1. First, determine the connectURL from the TD. Start with the cache.
	svc.cmux.Lock()
	connectURL, found := svc.connectURLByThingID[tdoc.ID]
	if connectURL == "" {
		// first request for this Thing.
		// Determine the connectURL from the TD and store it.
		connectURL, form, err = svc.GetConnectURL(tdoc, op, name)
		if err != nil {
			slog.Warn("GetClientConnection: Unable to determine a connectURL",
				"thingID", tdoc.ID, "op", op, "name", name)
			return nil, err
		}
		// cache the connectURL for fast lookup on the next request
		svc.connectURLByThingID[tdoc.ID] = connectURL
	}
	// load the client connection if it exists
	cl, found = svc.deviceConnections[connectURL]
	defer svc.cmux.Unlock()

	// 2. If a valid connection does not yet exist, establish one.
	if !found {

		// 3. Create a new client for the connect URL
		c, err = clients.NewTransportClientFromForm(tdoc, form, svc.rootCAs)
		if err != nil {
			return nil, err
		}
		c.SetTimeout(svc.GetTimeout())

		// if a client certificate is available set it for authentication
		// TBD: set a client cert per device? seems a bit overkill
		if svc.clientCert != nil {
			err = c.SetClientCert(svc.clientCert)
		}

		// Note-1: that while the TD form contains authentication instructions, the
		// available credentials determine the format used.
		//
		// Note-2: when connecting to a gateway with multiple devices, this requires
		// that the same credentials are set for every single thingID the device
		// offers, even though they all have the same origin and use the same
		// connection.
		// The credentials are therefore set per connectURL, not the thingID.
		//
		clientID, secret, secScheme, found := svc.credStore.GetCredentials(connectURL)
		if !found {
			clientID = svc.clientID
		}
		err = c.SetAuthToken(clientID, secret, secScheme)
		if err != nil {
			// No auth. Discard this connection.
			slog.Warn("GetClientConnection: failed", "ThingID", tdoc.ID, "err", err.Error())
			return nil, err
		}
		if svc.autoReconnect {
			// reconnect connects the client on start
			cl, err = reconnect_service.NewReconnectService(c)
		} else {
			// connect directly. Reconnect is not used.
			// TODO: without reconnect, should the connection be discarded?
			cl = c
		}
		svc.deviceConnections[connectURL] = cl

		// forward notifications to this service and up to its consumer
		cl.SetNotificationSink(svc)
		cl.Start()
	}

	return cl, err
}

// Return the reverse-client connection to a device, if it exists.
// This returns nil if the clientID does not have an existing connection.
func (svc *RouterServiceImpl) GetRCConnection(clientID string) (c api.IConnection) {
	if svc.getSrv == nil {
		return nil
	}
	serverList := svc.getSrv()
	for _, tp := range serverList {
		c := tp.GetConnectionByClientID(clientID)
		if c != nil {
			return c
		}
	}
	return nil
}

// HandleRequest handles requests or routes the request to its destination
func (svc *RouterServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var resp *msg.ResponseMessage

	if req.ThingID != svc.GetThingID() {
		return svc.RouteRequest(req, replyTo)
	}
	// handle requests for router itself
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
func (svc *RouterServiceImpl) HasCredentials(thingID string) (credType string, found bool) {
	// determine the connectURL for this thing
	tdoc := svc.getTD(thingID)
	if tdoc == nil {
		slog.Warn("DeleteCredentials: Cant delete credentials. TD for thingID not found", "thingID", thingID)
		return "", false
	}
	connectURL, _, err := svc.GetConnectURL(tdoc, "", "")
	if err != nil {
		return "", false
	}

	return svc.credStore.HasCredentials(connectURL)
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
//  2. If the TD contains an RC clientID, injected by the directory service, then lookup
//     the device's RC connection to the server and forward the request.
//  3. If the TD points to a non RC device then establish a connection or re-use
//     an existing connection from the pool.
func (svc *RouterServiceImpl) RouteRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// the requested thingID must be known
	tdoc := svc.getTD(req.ThingID)
	if tdoc == nil {
		// thingID not known
		// option 1: forward the request downstream
		// option 2: try GetRCConnection using thingID. RC's use their thingID as clientID
		//    this is a bit of a narrow use-case so don't bother for now.
		err = svc.ForwardRequest(req, replyTo)
		if err != nil {
			err = fmt.Errorf("RouteRequest: TD not found for thing '%s' and forwarding request failed: %w",
				req.ThingID, err)
			// just log as info as this can be legit.
			slog.Info("RouteRequest", "err", err.Error())
		}
		return err
	}

	// if the tdoc has an RC clientID then look for its RC connection
	rcClientID := tdoc.GetRCClientID()
	if rcClientID != "" {
		c := svc.GetRCConnection(rcClientID)
		if c == nil {
			err = fmt.Errorf("RouteRequest: device '%s' isnt connected", rcClientID)
		} else {
			err = c.SendRequest(req, replyTo)
		}
	} else {
		c, err2 := svc.GetClientConnection(tdoc, req.Operation, req.Name)
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

// Enable/disable auto reconnect for new connections
func (svc *RouterServiceImpl) SetAutoReconnect(enable bool) {
	svc.autoReconnect = enable
}

// Provide client certificate for authentication of new client connections
func (svc *RouterServiceImpl) SetClientCert(clientCert *tls.Certificate) {
	svc.clientCert = clientCert
}

// Stop the router service.
// This closes all established client connections.
func (svc *RouterServiceImpl) Stop() {
	slog.Info("Stop: Stopping router service")
	for clientID, c := range svc.deviceConnections {
		_ = clientID
		c.Stop()
	}
	svc.deviceConnections = nil
	// last close credential store
	svc.credStore.Close()
}

// NewRouterServiceImpl creates a new router service
//
// Use getSrv if routing requests to server RC connected device should be supported.
// AutoReconnect will attempt to automatically reconnect failed client connections. Note that this
// might hide authentication problems.
//
//	storageDir for the credentials storage directory, "" for in-memory testing
//	autoReconnect flag, to enable auto-reconnect on client connections
//	clientID default clientID to connect to devices with
//	clientCert optional client certificate to connect to devices with - overrides clientID
//	rootCAs are the CA's used to verify TLS connections to devices
//	timeout is the maximum communication timeout with connect clients
//	getTD  handler to lookup a TD for a thingID from a directory. Required.
//	getSrv handler returning a list of transport servers that can contain RC devices.
func NewRouterServiceImpl(
	storageDir string,
	autoReconnect bool,
	clientID string,
	clientCert *tls.Certificate,
	rootCAs *x509.CertPool,
	timeout time.Duration,
	getTD func(thingID string) *td.TD,
	getSrv func() []api.ITransportServer,
) (*RouterServiceImpl, error) {

	var storageFile string
	if getTD == nil {
		return nil, fmt.Errorf("NewRouterServiceImpl: missing getTD provider")
	}

	slog.Info("Start: Starting router service")
	if timeout == 0 {
		timeout = msg.DefaultRnRTimeout
	}

	if storageDir != "" {
		fileName := "deviceCredentials.json"
		storageFile = filepath.Join(storageDir, fileName)
	}
	credStore := NewCredentialsStore(storageFile)
	err := credStore.Open()

	thingID := router.RouterCellType + "-" + shortid.MustGenerate()
	svc := &RouterServiceImpl{
		HiveCellBase:      cells.NewHiveCellBase(thingID, timeout),
		autoReconnect:     autoReconnect,
		clientID:          clientID,
		clientCert:        clientCert,
		credStore:         credStore,
		rootCAs:           rootCAs,
		getTD:             getTD,
		preferredProtocol: api.WotWebsocketProtocolType,
		storageFile:       storageFile,
		getSrv:            getSrv,
		// connections by connectURL
		deviceConnections:   make(map[string]api.IHiveCell),
		connectURLByThingID: make(map[string]string),
	}

	var _ router.IRouterService = svc // interface check

	return svc, err
}
