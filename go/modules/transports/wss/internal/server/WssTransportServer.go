package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	wssapi "github.com/hiveot/hivekit/go/modules/transports/wss/api"
	wssencoder "github.com/hiveot/hivekit/go/modules/transports/wss/internal/encoder"
	"github.com/hiveot/hivekit/go/utils"
)

// WssTransportServer is a transport module that serves Websocket connections over http.
// This implements both ITransportServer and IHiveModule interfaces.
type WssTransportServer struct {
	transports.TransportServerBase

	// actual server exposing routes including websocket endpoint
	httpServer transports.IHttpServer

	// Websocket protocol message converter
	encoder transports.IMessageEncoder // WoT or Hiveot message format

	// the time to wait for responses to request
	respTimeout time.Duration

	// WoT or Hiveot subprotocol
	// subprotocol string

	// listening path for incoming connections
	wssPath string
}

// ServeWssConnection serves a new websocket connection.
// This creates an instance of the HiveotWSSConnection handler for reading and
// writing messages.
//
// This doesn't return until the connection is closed by either client or server.
//
// serverRequestHandler and serverResponseHandler are used as handlers for incoming
// messages.
func (m *WssTransportServer) ServeWssConnection(w http.ResponseWriter, r *http.Request) {
	//An active session is required before accepting the request. This is created on
	//authentication/login. Until then connections are blocked.
	// rp, err := m.httpServer.GetRequestParams(r)
	// if err != nil {
	// net.WriteError(w, err, 0)
	// }
	clientID, err := m.httpServer.GetClientIdFromContext(r)
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

	// the new server connection sends messages to the module sink
	c := NewWSSServerConnection(clientID, r, wssConn, m.encoder,
		m.ForwardRequest, m.ForwardNotification)
	c.SetTimeout(m.respTimeout)
	// add connection sends a notification
	err = m.AddConnection(c)

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
	m.RemoveConnection(c)
	// if m.connectHandler != nil {
	// m.connectHandler(false, c, nil)
	// }
}

// Start listening for incoming websocket connections
//
//	yamlConfig: todo, wssPath
func (m *WssTransportServer) Start() (err error) {

	connectURL := m.httpServer.GetConnectURL()
	slog.Info("Start: Starting websocket transport server, Listening on: " + connectURL)

	// create routes
	router := m.httpServer.GetProtectedRoute()
	router.Get(m.wssPath, m.ServeWssConnection)
	return nil
}

// Stop disconnects clients and remove connection listening
func (m *WssTransportServer) Stop() {
	slog.Info("Stop: Stopping websocket transport server")
	m.CloseAll()
	router := m.httpServer.GetProtectedRoute()
	router.Delete(m.wssPath, m.ServeWssConnection)
}

// NewHiveotWssTransportServer creates a websocket server module using serving HiveOT websocket
// connections from consumers and agents.
//
// httpServer is the http server the websocket is using
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewHiveotWssServer(httpServer transports.IHttpServer, respTimeout time.Duration) *WssTransportServer {

	httpURL := httpServer.GetConnectURL()
	urlParts, err := url.Parse(httpURL)
	if err != nil {
		panic("NewHiveotWssModule: Http server has invalid URL")
	}

	if respTimeout == 0 {
		respTimeout = transports.DefaultRpcTimeout
	}
	m := &WssTransportServer{
		httpServer: httpServer,
		encoder:    transports.NewRRNJsonEncoder(),
		// connectHandler: nil,
		respTimeout: respTimeout,
		wssPath:     wssapi.HiveotWebsocketPath,
	}
	// set the base parameters
	connectURL := fmt.Sprintf("%s://%s%s", transports.UriSchemeHiveotWebsocket, urlParts.Host, m.wssPath)
	m.Init(
		wssapi.HiveotWebsocketModuleType,
		transports.ProtocolTypeHiveotWebsocket,
		transports.SubprotocolHiveotWebsocket,
		connectURL, httpServer.GetAuthenticator())
	return m
}

// Create a websocket module using WoT messaging format
// This uses the WoT websocket protocol message converter to convert between
// the standard RRN messages and the WoT websocket message format.
//
// httpServer is the http server the websocket is using
// respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewWotWssServer(httpServer transports.IHttpServer, respTimeout time.Duration) *WssTransportServer {
	httpURL := httpServer.GetConnectURL()
	urlParts, err := url.Parse(httpURL)
	if err != nil {
		panic("NewWotWssModule: Http server has invalid URL")
	}
	if respTimeout == 0 {
		respTimeout = transports.DefaultRpcTimeout
	}
	m := &WssTransportServer{
		httpServer:  httpServer,
		encoder:     wssencoder.NewWotWssMsgEncoder(),
		respTimeout: respTimeout,
		wssPath:     wssapi.WotWebsocketPath,
	}

	connectURL := fmt.Sprintf("%s://%s%s", transports.UriSchemeWotWebsocket, urlParts.Host, m.wssPath)
	m.Init(
		wssapi.HiveotWebsocketModuleType,
		transports.ProtocolTypeWotWebsocket,
		transports.SubprotocolWotWebsocket,
		connectURL, httpServer.GetAuthenticator())
	// m.UpdateProperty(transports.PropName_NrConnections, 0)

	var _ modules.IHiveModule = m         // interface check
	var _ transports.ITransportServer = m // interface check

	return m
}
