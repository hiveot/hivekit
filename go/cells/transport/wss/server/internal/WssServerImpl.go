package internal

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/cells/transport"
	"github.com/hiveot/hivekit/go/cells/transport/wss"
	"github.com/hiveot/hivekit/go/cells/transport/wss/msgconverter"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/teris-io/shortid"
)

// WssServerImpl is a transport server that serves Websocket connections over http.
// This implements both ITransportServer and IHiveCell interfaces.
type WssServerImpl struct {
	*transport.TransportServerBase

	// flag, include affordances when adding forms
	includeAffordances bool

	// actual server exposing routes including websocket endpoint
	httpServer api.IHttpServer

	// Websocket protocol message converter
	encoder transport.IMessageEncoder // WoT or Hiveot message format

	// the time to wait for responses to request
	respTimeout time.Duration

	// serverTD is the TD describing how to connect to this server
	serverTD *td.TD

	// WoT or Hiveot subprotocol
	subprotocol string

	// listening path for incoming connections
	wssPath string
}

// AddTDSecForms updates the TD with base URI, security scheme and forms for use of
// this protocol to the given TD.
//
// Since the contentType is the default application/json it is omitted
//
// 'includeAffordances' adds forms to all affordances to be compliant with the specifications.
// Btw, this is a waste of space in the TD as it required but not needed with some protocols.
func (srv *WssServerImpl) AddTDSecForms(tdoc *td.TD) {
	// 1. Add the base connection endpoint
	// TODO: if this Thing supports multiple protocols it might conflict with
	// the base. In that case base cannot be used and all hrefs must be absolute?
	href := srv.GetConnectURL()
	tdoc.Base = href
	subprotocol := srv.subprotocol

	// 2. Set the security scheme used by the authenticator.
	// TODO: risk of duplicates?
	authr := srv.httpServer.GetAuthenticator()
	authr.AddSecurityScheme(tdoc)

	// 3. add top level form for thing level  operations
	// the href is the connection URL because it is the same as base for all forms in this protocol
	form := td.NewForm("", srv.GetConnectURL())
	form.SetSubprotocol(subprotocol)
	form["op"] = []string{
		td.HTOpPing,
		td.OpInvokeAction, td.OpCancelAction,
		td.OpQueryAction, td.OpQueryAllActions,

		td.OpReadProperty, td.OpReadAllProperties, td.OpReadMultipleProperties,
		td.OpWriteProperty, td.OpWriteMultipleProperties,
		td.OpObserveProperty, td.OpObserveAllProperties, td.OpObserveMultipleProperties,
		td.OpUnobserveProperty, td.OpUnobserveAllProperties, td.OpUnobserveMultipleProperties,

		// hiveot supports reading latest events
		td.HTOpReadEvent, td.HTOpReadAllEvents,
		td.OpSubscribeEvent, td.OpSubscribeAllEvents,
		td.OpUnsubscribeEvent, td.OpUnsubscribeAllEvents,
	}
	//form["contentType"] = "application/json"
	tdoc.Forms = append(tdoc.Forms, form)

	// 4. Add forms to all affordances to be compliant with the specifications.
	// This does uses the same href to prevent conflict with multiple protocols
	if srv.includeAffordances {

		for _, aff := range tdoc.Actions {
			form := aff.AddForm("", href, "", nil)
			form.SetSubprotocol(subprotocol)
			form["op"] = []string{td.OpInvokeAction, td.OpQueryAction}
		}
		for _, aff := range tdoc.Events {
			form := aff.AddForm("", href, "", nil)
			form.SetSubprotocol(subprotocol)
			form["op"] = []string{td.HTOpReadEvent, td.OpSubscribeEvent, td.OpUnsubscribeEvent}
		}
		for _, aff := range tdoc.Properties {
			form := aff.AddForm("", href, "", nil)
			form.SetSubprotocol(subprotocol)
			if !aff.WriteOnly {
				form["op"] = []string{td.OpReadProperty, td.OpObserveProperty, td.OpUnobserveProperty}
			}
			if !aff.ReadOnly {
				form["op"] = []string{td.OpWriteProperty}
			}
			form["op"] = []string{td.HTOpReadEvent, td.OpSubscribeEvent, td.OpUnsubscribeEvent}
		}
	}
}

// GetTD returns the server TD, containing connection and authentication information
func (srv *WssServerImpl) GetTD() *td.TD {
	return srv.serverTD
}

// ServeWssConnection serves a new websocket connection.
// This creates an instance of the HiveotWSSConnection handler for reading and
// writing messages.
//
// This doesn't return until the connection is closed by either client or server.
//
// serverRequestHandler and serverResponseHandler are used as handlers for incoming
// messages.
func (srv *WssServerImpl) ServeWssConnection(w http.ResponseWriter, r *http.Request) {
	//An active session is required before accepting the request. This is created on
	//authentication/login. Until then connections are blocked.
	// rp, err := m.httpServer.GetRequestParams(r)
	// if err != nil {
	// net.WriteError(w, err, 0)
	// }
	clientID, err := srv.httpServer.GetClientIdFromContext(r)
	if err != nil {
		utils.WriteError(w, err, 0)
	}
	slog.Info("Serve: Receiving Websocket connection",
		slog.String("clientID", clientID),
	)

	if err != nil {
		slog.Error("Serve. No clientID",
			"remoteAddr", r.RemoteAddr)
		errMsg := "no auth session available. Login first."
		http.Error(w, errMsg, http.StatusUnauthorized)
		return
	}

	// upgrade and validate the connection
	var upgrader = websocket.Upgrader{} // use default options
	wssConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Serve: Connection upgrade failed",
			"clientID", clientID, "err", err.Error())
		return
	}

	// the new server connection sends messages to the cell sink
	c := NewWSSServerConnection(clientID, r, wssConn, srv.encoder,
		srv.EmitRequest, srv.EmitNotification)
	c.SetTimeout(srv.respTimeout)
	// add connection sends a notification
	err = srv.AddConnection(c)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	// don't return until the connection is closed
	c.ReadLoop(r.Context(), wssConn)

	// if this fails then the connection is already closed (CloseAll)
	err = wssConn.Close()

	_ = err
	// finally cleanup the connection
	srv.RemoveConnection(c)

}

// Stop disconnects clients and remove connection listening
func (srv *WssServerImpl) Stop() {
	slog.Info("Stop: Stopping websocket transport server")
	srv.CloseAll()
	router := srv.httpServer.GetProtectedRoute()
	router.Delete(srv.wssPath, srv.ServeWssConnection)
}

// NewHiveotWssServerImpl starts a ready-to-use websocket server for serving
// HiveOT websocket connections from consumers and devices.
//
// httpServer is the http server the websocket is using.
// This immediately starts listening for connections.
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by devices.
func NewHiveotWssServerImpl(
	httpServer api.IHttpServer, respTimeout time.Duration) (*WssServerImpl, error) {

	if httpServer == nil {
		err := fmt.Errorf("NewWotWssServerImpl: Http server is nil")
		return nil, err
	}
	httpURL := httpServer.GetConnectURL()
	urlParts, err := url.Parse(httpURL)
	if err != nil {
		err = fmt.Errorf("StartHiveotWssServerImpl: Http server has invalid URL: %w", err)
		return nil, err
	}

	if respTimeout == 0 {
		respTimeout = msg.DefaultRnRTimeout
	}
	thingID := wss.HiveotWebsocketServerCellType + "-" + shortid.MustGenerate()

	connectURL := fmt.Sprintf("%s://%s%s",
		api.HiveotWebsocketScheme, urlParts.Host, wss.HiveotWebsocketPath)

	slog.Info("Start: Starting HiveOT websocket transport server, Listening on: " + connectURL)

	serverTD := td.NewTD(thingID, "HiveOT Websocket server", vocab.DeviceTypeService)
	authenticator := httpServer.GetAuthenticator()
	srv := &WssServerImpl{
		TransportServerBase: transport.NewTransportServerBase(thingID, connectURL, authenticator),

		encoder: transport.NewRRNJsonEncoder(),

		includeAffordances: false, // forms per affordance for websockets are a waste of space

		httpServer: httpServer,
		// connectHandler: nil,
		respTimeout: respTimeout,
		serverTD:    serverTD,
		subprotocol: api.HiveotWebsocketSubprotocol,
		wssPath:     wss.HiveotWebsocketPath,
	}

	// create routes
	router := httpServer.GetProtectedRoute()
	router.Get(srv.wssPath, srv.ServeWssConnection)

	// create a TD describing this server along with its connection URL
	srv.AddTDSecForms(srv.serverTD)
	return srv, err
}

// NewWotWssServerImpl starts a ready-to-use websocket transport server using WoT
// messaging format.
//
// This uses the WoT websocket protocol message converter to convert between
// the standard RRN messages and the WoT websocket message format.
//
// This immediately starts listening for connections.
//
// httpServer is the http server the websocket is using
// respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by devices.
func NewWotWssServerImpl(
	httpServer api.IHttpServer, respTimeout time.Duration) (*WssServerImpl, error) {

	if httpServer == nil {
		return nil, fmt.Errorf("StartWotWssServerImpl: Http server is nil")
	}
	httpURL := httpServer.GetConnectURL()
	urlParts, err := url.Parse(httpURL)
	if err != nil {
		return nil, fmt.Errorf("StartWotWssServerImpl: Http server has invalid URL")
	}
	if respTimeout == 0 {
		respTimeout = msg.DefaultRnRTimeout
	}
	thingID := wss.WotWebsocketServerCellType + "-" + shortid.MustGenerate()

	connectURL := fmt.Sprintf("%s://%s%s", api.WotWebsocketScheme, urlParts.Host, wss.WotWebsocketPath)
	slog.Info("StartWotWssServerImpl: Starting WoT websocket transport server, Listening on: " + connectURL)

	serverTD := td.NewTD(thingID, "WoT Websocket server", vocab.DeviceTypeService)
	authenticator := httpServer.GetAuthenticator()
	srv := &WssServerImpl{
		TransportServerBase: transport.NewTransportServerBase(thingID, connectURL, authenticator),

		includeAffordances: false, // forms per affordance for websockets are a waste of space

		httpServer:  httpServer,
		encoder:     msgconverter.NewWotWssMsgEncoder(),
		respTimeout: respTimeout,
		serverTD:    serverTD,
		wssPath:     wss.WotWebsocketPath,
		subprotocol: api.WotWebsocketSubprotocol,
	}

	// create routes
	router := httpServer.GetProtectedRoute()
	router.Get(srv.wssPath, srv.ServeWssConnection)

	// create a TD describing this server along with its connection URL
	srv.AddTDSecForms(srv.serverTD)
	var _ api.IHiveCell = srv        // interface check
	var _ api.ITransportServer = srv // interface check

	return srv, nil
}
