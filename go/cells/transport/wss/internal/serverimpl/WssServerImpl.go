package serverimpl

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
	"github.com/hiveot/hivekit/go/cells/transport/wss/internal"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/teris-io/shortid"
)

// WssServerImpl is a transport server that serves Websocket connections over http.
// This implements both ITransportServer and IHiveCell interfaces.
type WssServerImpl struct {
	*transport.TransportServerBase

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
		srv.ForwardRequest, srv.ForwardNotification)
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
	// if m.connectHandler != nil {
	// m.connectHandler(false, c, nil)
	// }
}

// Start listening for incoming websocket connections
//
//	yamlConfig: todo, wssPath
func (srv *WssServerImpl) Start() (err error) {

	connectURL := srv.httpServer.GetConnectURL()
	slog.Info("Start: Starting websocket transport server, Listening on: " + connectURL)

	// create routes
	router := srv.httpServer.GetProtectedRoute()
	router.Get(srv.wssPath, srv.ServeWssConnection)

	// create a TD describing this server along with its connection URL
	thingID := srv.GetThingID()
	srv.serverTD = td.NewTD(thingID, srv.subprotocol+" Websocket server", vocab.DeviceTypeService)
	srv.AddTDSecForms(srv.serverTD, false)
	return err
}

// Stop disconnects clients and remove connection listening
func (srv *WssServerImpl) Stop() {
	slog.Info("Stop: Stopping websocket transport server")
	srv.CloseAll()
	router := srv.httpServer.GetProtectedRoute()
	router.Delete(srv.wssPath, srv.ServeWssConnection)
}

// NewHiveotWssTransportServer creates a websocket transport server using serving HiveOT websocket
// connections from consumers and devices.
//
// httpServer is the http server the websocket is using
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by devices.
func NewHiveotWssServerImpl(httpServer api.IHttpServer, respTimeout time.Duration) *WssServerImpl {

	httpURL := httpServer.GetConnectURL()
	urlParts, err := url.Parse(httpURL)
	if err != nil {
		panic("NewHiveotWssServerImpl: Http server has invalid URL")
	}

	if respTimeout == 0 {
		respTimeout = msg.DefaultRnRTimeout
	}
	thingID := wss.HiveotWebsocketServerCellType + "-" + shortid.MustGenerate()
	connectURL := fmt.Sprintf("%s://%s%s",
		api.HiveotWebsocketScheme, urlParts.Host, wss.HiveotWebsocketPath)
	authenticator := httpServer.GetAuthenticator()
	m := &WssServerImpl{
		TransportServerBase: transport.NewTransportServerBase(thingID, connectURL, authenticator),

		encoder:    transport.NewRRNJsonEncoder(),
		httpServer: httpServer,
		// connectHandler: nil,
		respTimeout: respTimeout,
		subprotocol: api.HiveotWebsocketSubprotocol,
		wssPath:     wss.HiveotWebsocketPath,
	}
	return m
}

// Create a websocket transport server using WoT messaging format.
// This uses the WoT websocket protocol message converter to convert between
// the standard RRN messages and the WoT websocket message format.
//
// httpServer is the http server the websocket is using
// respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by devices.
func NewWotWssServerImpl(httpServer api.IHttpServer, respTimeout time.Duration) *WssServerImpl {
	if httpServer == nil {
		panic("NewWotWssServerImpl: Http server is nil")
	}
	httpURL := httpServer.GetConnectURL()
	urlParts, err := url.Parse(httpURL)
	if err != nil {
		panic("NewWotWssServerImpl: Http server has invalid URL")
	}
	if respTimeout == 0 {
		respTimeout = msg.DefaultRnRTimeout
	}
	thingID := wss.WotWebsocketServerCellType + "-" + shortid.MustGenerate()
	connectURL := fmt.Sprintf("%s://%s%s", api.WotWebsocketScheme, urlParts.Host, wss.WotWebsocketPath)
	authenticator := httpServer.GetAuthenticator()
	m := &WssServerImpl{
		TransportServerBase: transport.NewTransportServerBase(thingID, connectURL, authenticator),

		httpServer:  httpServer,
		encoder:     internal.NewWotWssMsgEncoder(),
		respTimeout: respTimeout,
		wssPath:     wss.WotWebsocketPath,
		subprotocol: api.WotWebsocketSubprotocol,
	}

	var _ api.IHiveCell = m        // interface check
	var _ api.ITransportServer = m // interface check

	return m
}
