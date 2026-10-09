package internal

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	reconnect_service "github.com/hiveot/hivekit/go/cells/reconnect/service"
	"github.com/hiveot/hivekit/go/cells/router"
	"github.com/hiveot/hivekit/go/cells/transport/clients"
)

// for caching connection URLs with related form used.
type connectURLForm struct {
	ConnectURL string
	Form       *td.Form
	TDUpdated  string
}

// RouterServiceImpl implements request routing to stand-alone devices.
// Use-cases:
//
//	a. Consumer side;
//		1. router connect to stand-alone device, using directory client:
//		   	* getTD provides a valid TD. Without it the router fails.
//			* getTD is typically a directory client, that needs a TDD
//		    * discovery can provide a TDD to the directory.
//		2. router connects to SA device using discovery client:
//		   	* getTD provides a valid TD. Without it the router fails.
//		    * getTD is a discovery client that searches for SA devices
//	b. Server side, inside a gateway
//		3. router connects to stand-alone device using directory client
//	    	* same as 1.
//
//	c. Note that a RC device doesnt need a router, just discovery and a gateway
//	   client. So, all this is just to support stand-alone devices.
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
	connectURLByThingID map[string]connectURLForm

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

	// the preferred protocol to use when creating a new client connection
	preferredProtocol string

	// The root CA certificates used to verify device connections
	rootCAs *x509.CertPool

	// location of the device credentials store. "" for in-memory only.
	storageDir string
}

// Add the secret to access a Thing.
//
// if thingID is empty then the credentials are used for all devices for which
// no credentials are set. Use with care as it exposes the token to these devices.
func (svc *RouterServiceImpl) AddCredentials(
	thingID string, clientID string, secret string, credType string) (err error) {

	creds := ConnectCredentials{
		ClientID: clientID,
		Secret:   secret,
		CredType: credType,
	}

	// Set as default credentials if no thingID is provided.
	if thingID == "" {
		err = svc.credStore.AddCredentials("", creds)
		return
	}

	// determine the connectURL for this thing
	tdoc := svc.getTD(thingID)
	if tdoc == nil {
		slog.Error("AddDeviceCredential: Cant add credentials. TD for thingID not found", "thingID", thingID)
	} else {
		connectURL, _, err := svc.GetConnectURL(tdoc, "", "")
		if err == nil {
			err = svc.credStore.AddCredentials(connectURL, creds)
		}
	}
	return err
}

// Remove the secret to access a Thing
func (svc *RouterServiceImpl) DeleteCredentials(thingID string) error {
	// determine the connectURL for this thing
	tdoc := svc.getTD(thingID)
	if tdoc == nil {
		err := fmt.Errorf("DeleteCredentials: Cant delete credentials. TD for thingID '%s' not found", thingID)
		return err
	}
	connectURL, _, err := svc.GetConnectURL(tdoc, "", "")
	if err != nil {
		err = fmt.Errorf("DeleteCredentials: TD '%s' has no connection information", thingID)
		return err
	}
	err = svc.credStore.DeleteCredentials(connectURL)
	return err
}

// Determine the connection URL from the Thing TD and operation
//
// This caches previous lookup based on the thingID. If no URL is known it is
// created using the TD forms. In order of preference: websocket first
//
// NOTE: this has the limitation that only a single client per ThingID is used.
// The first usage determines this connection. Successive requests for other operations
// return the same connectionURL.
//
//	tdoc is the TD to get the URL from
//	op is the operation to use
//	name is the optional affordance name
//
// This returns the connect URL, the form used or an error
func (svc *RouterServiceImpl) GetConnectURL(
	tdoc *td.TD, op string, name string) (connectURL string, connectForm *td.Form, err error) {
	var hrefURL *url.URL
	var match bool

	// cache previous lookups
	svc.cmux.RLock()
	cuf, found := svc.connectURLByThingID[tdoc.ID]
	svc.cmux.RUnlock()
	if found && cuf.TDUpdated == tdoc.Modified {
		return cuf.ConnectURL, cuf.Form, nil
	}

	// Try to find a form using the preferred protocol first
	prefScheme := api.WotWebsocketScheme
	prefSubprotocol := api.WotWebsocketSubprotocol
	// Note: the definition of protocol-type is scheme:subprotocol
	protocolParts := strings.Split(svc.preferredProtocol, ":")
	if len(protocolParts) > 1 {
		prefScheme = protocolParts[0]
		prefSubprotocol = protocolParts[1]
	}
	// Attempt to find a form matching the preferred protocol
	connectForm, match = tdoc.GetForm(op, name, prefScheme, prefSubprotocol)
	_ = match
	if connectForm != nil {

		// get the full URL for the operation
		hrefURL, err = connectForm.ResolveHRef(tdoc.Base, nil)

		// need an href
		if err == nil {

			// determine the connectURL that identifies the client connection
			urlScheme := strings.ToLower(hrefURL.Scheme)
			if urlScheme == "https" {
				// ignore the path as it differs per operation/name but can use the same client
				connectURL = fmt.Sprintf("%s://%s", hrefURL.Scheme, hrefURL.Host)
			} else {
				// use the full URL as the connection endpoint
				connectURL = hrefURL.String()
			}
		}
	}
	// Store the result of this lookup for the cache regardless of the outcome.
	svc.cmux.Lock()
	cuf = connectURLForm{
		ConnectURL: connectURL,
		Form:       connectForm,
		TDUpdated:  tdoc.Modified,
	}
	svc.connectURLByThingID[tdoc.ID] = cuf
	svc.cmux.Unlock()

	return connectURL, connectForm, nil
}

// GetClientConnection returns a client for sending requests to the server with
// the given connection URL. If a connection doesn't exists then create it.
//
// Previous connections are re-used. The connect URL identifies the connection.
// For each requested thingID, the connectURL is calculated and cached.
//
// If the 'autoReconnect' option is configured then this wraps the connection in a
// Reconnect client that automatically reconnects and resubscribes if the connection
// fails.
//
//	tdoc is the TD of the device to connect to.
//	op is the operation to perform
//	name is the optional affordance name for the operation. "" for thing level operations.
func (svc *RouterServiceImpl) GetClientConnection(
	tdoc *td.TD, connectURL string, cform *td.Form) (cl api.IHiveCell, err error) {

	var c api.ITransportClient
	// var form *td.Form

	// 1. load the client connection if it exists
	svc.cmux.Lock()
	cl, found := svc.deviceConnections[connectURL]
	defer svc.cmux.Unlock()

	// 2. If a valid connection does not yet exist, establish one, if possible.
	if !found {

		c, err = clients.NewTransportClientFromForm(tdoc, cform, svc.rootCAs)
		if err != nil {
			return nil, err
		}
		c.SetTimeout(svc.GetTimeout())

		// if a client certificate is available set it for authentication
		if svc.clientCert != nil {
			err = c.SetClientCert(svc.clientCert)
		}

		// Note-1: that while the TD form contains authentication instructions, the
		// available credentials determine the format used.
		//
		// Note-2: when connecting to a gateway with multiple devices, this requires
		// that the same credentials are set for every single thingID the device
		// offers, even though they all use the same connection.
		// The credentials are therefore set per connectURL, not the thingID.
		//
		// Note-3: if the connect URL is that of the gateway that runs this router then this can
		// cause a loop, resulting in a failed request since RnR for request already
		// exists.
		// TODO: identify this situation and return with an error

		// determine who to identify as for this connection; fall back to default
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
			cl.SetTimeout(svc.GetTimeout())
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

// HandleRequest handles requests or routes the request to its destination
func (svc *RouterServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var resp *msg.ResponseMessage
	if req.ThingID == "" {
		// unable to route, pass it on
		return svc.ForwardRequest(req, replyTo)
	} else if req.ThingID != svc.GetID() {
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

// Route the request to remote devices.
// If no connection can be established then pass the request to the sink.
//
// Lookup the TD of the ThingID and determine its destination:
func (svc *RouterServiceImpl) RouteRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// 1. the requested thingID must be known
	tdoc := svc.getTD(req.ThingID)
	if tdoc == nil {
		// without a TD there is nothing the router can do here.
		err = svc.ForwardRequest(req, replyTo)
		if err != nil {
			err = fmt.Errorf(
				"RouteRequest: TD not found for thing '%s' and forwarding request failed: %w",
				req.ThingID, err)
			slog.Warn(err.Error())
		}
		return err
	}

	// 2. this is a RC device. Can't connect to it. Should not have come here.
	rcid := tdoc.GetRCID()
	if rcid != "" {
		err := fmt.Errorf("RouteRequest: invalid attempt to connect to a RC device '%s'. Should not have come here.", rcid)
		slog.Error(err.Error())
		return err
	}

	// 2. the connection URL is needed for establishing a device connection
	connectURL, connectForm, err := svc.GetConnectURL(tdoc, req.Operation, req.Name)
	if connectURL != "" {
		c, err2 := svc.GetClientConnection(tdoc, connectURL, connectForm)
		if c == nil {
			slog.Warn("RouteRequest: Unable to establish a connection to client", "err", err2)
			err = err2
		} else if err2 != nil {
			slog.Warn("RouteRequest: Connection failed", "err", err2)
			err = err2
		} else {
			err = c.HandleRequest(req, replyTo)
		}
		return err
	}

	// 3. if the TD has no connection then pass the request to the sink.
	// See also the rcservice for use in gateways that can pass the request to
	// a reverse-connected client.

	err = svc.ForwardRequest(req, replyTo)

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
//	getTD  handler to lookup a TD for a thingID from a directory. Required.
func NewRouterServiceImpl(
	storageDir string,
	autoReconnect bool,
	clientID string,
	clientCert *tls.Certificate,
	rootCAs *x509.CertPool,
	getTD func(thingID string) *td.TD,
) (*RouterServiceImpl, error) {

	if getTD == nil {
		return nil, fmt.Errorf("NewRouterServiceImpl: missing getTD provider")
	}

	slog.Info("NewRouterServiceImpl: Starting router service")

	credStore := NewCredentialsStore(storageDir)
	err := credStore.Open()

	thingID := router.RouterDefaultThingID
	svc := &RouterServiceImpl{
		HiveCellBase:      cells.NewHiveCellBase(thingID),
		autoReconnect:     autoReconnect,
		clientID:          clientID,
		clientCert:        clientCert,
		credStore:         credStore,
		rootCAs:           rootCAs,
		getTD:             getTD,
		preferredProtocol: api.WotWebsocketProtocolType,
		storageDir:        storageDir,
		// connections by connectURL
		deviceConnections:   make(map[string]api.IHiveCell),
		connectURLByThingID: make(map[string]connectURLForm),
	}

	var _ router.IRouterService = svc // interface check

	return svc, err
}
