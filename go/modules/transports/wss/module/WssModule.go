package module

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	"github.com/hiveot/hivekit/go/modules/transports/httptransport"
	"github.com/hiveot/hivekit/go/modules/transports/wss"
	wssapi "github.com/hiveot/hivekit/go/modules/transports/wss/api"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
)

// WssModule is a transport module that serves Websocket connections over http.
type WssModule struct {
	transports.TransportModuleBase
	// this handles request for this module
	msgAPI *WssRrnHandler

	// actual server exposing routes including websocket endpoint
	httpServer httptransport.IHttpServer

	// Websocket protocol message converter
	msgConverter transports.IMessageConverter // WoT or Hiveot message format

	// registered handler to notify of incoming connections
	serverConnectHandler transports.ConnectionHandler

	//
	subprotocol string // WoT or Hiveot subprotocol
	// listening path for incoming connections
	wssPath string
}

// // GetProtocolType returns the protocol type of this server
// func (srv *WssModule) GetProtocolType() string {
// 	return transports.ProtocolTypeWotWSS
// }

// HandleRequest passes the module request messages to the API handler.
// This has nothing to do with receiving requests over websockets.
func (m *WssModule) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// first attempt to procss the when targeted at this module
	if req.ThingID == m.GetModuleID() {
		err = m.msgAPI.HandleRequest(req, replyTo)
	}
	// if the request failed, then forward the request through the chain
	// the module base handles operations for reading properties
	if err != nil {
		err = m.HiveModuleBase.HandleRequest(req, replyTo)
	}
	return err
}

// Serve a new websocket connection.
// This creates an instance of the HiveotWSSConnection handler for reading and
// writing messages.
//
// This doesn't return until the connection is closed by either client or server.
//
// serverRequestHandler and serverResponseHandler are used as handlers for incoming
// messages.
func (m *WssModule) Serve(w http.ResponseWriter, r *http.Request) {
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
	slog.Info("Receiving Websocket connection", slog.String("clientID", clientID))

	if err != nil {
		slog.Warn("Serve. No clientID",
			"remoteAddr", r.RemoteAddr)
		errMsg := "no auth session available. Login first."
		http.Error(w, errMsg, http.StatusUnauthorized)
		return
	}

	// upgrade and validate the connection
	var upgrader = websocket.Upgrader{} // use default options
	wssConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("Serve. Connection upgrade failed",
			"clientID", clientID, "err", err.Error())
		return
	}

	// the new server connection sends messages to the module sink
	c := NewWSSServerConnection(clientID, r, wssConn, m.msgConverter, m.GetSink())

	err = m.AddConnection(c)

	if m.serverConnectHandler != nil {
		m.serverConnectHandler(true, nil, c)
	}
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
	if m.serverConnectHandler != nil {
		m.serverConnectHandler(false, nil, c)
	}
}

// Start listening for incoming websocket connections
func (m *WssModule) Start() (err error) {
	slog.Info("Starting websocket module, Listening on: " + m.GetConnectURL())

	if m.GetSink() == nil {
		err = fmt.Errorf("This Wss server module has no sink and will not work. Bye bye")
		return err
	}

	// TODO: detect if already listening
	err = m.TransportModuleBase.Start()
	// create routes
	router := m.httpServer.GetProtectedRoute()
	router.Get(m.wssPath, m.Serve)

	// The basic msg handler converts incoming module requests messages to the module API.
	// This has nothing to do with the http server.
	if err == nil {
		m.msgAPI = NewWssRrnHandler(m)
	}
	return nil
}

// Stop disconnects clients and remove connection listening
func (m *WssModule) Stop() {
	slog.Info("Stopping websocket module")
	m.CloseAll()
	router := m.httpServer.GetProtectedRoute()
	router.Delete(m.wssPath, m.Serve)
}

// Create a websocket module using hiveot RRN direct messaging
// This is used for agents that use reverse connections.
// This uses a passthrough message converter for requests, response and notifications
//
// httpServer is the http server the websocket is using
// sink is the optional receiver of request, response and notification messages, nil to set later
func NewHiveotWssModule(httpServer httptransport.IHttpServer, sink modules.IHiveModule) *WssModule {

	httpURL := httpServer.GetConnectURL()
	urlParts, err := url.Parse(httpURL)
	if err != nil {
		panic("NewHiveotWssModule: Http server has invalid URL")
	}

	m := &WssModule{
		httpServer:           httpServer,
		msgConverter:         direct.NewPassthroughMessageConverter(),
		subprotocol:          wss.SubprotocolHiveotWSS,
		serverConnectHandler: nil,
		wssPath:              wss.DefaultHiveotWssPath,
	}
	moduleID := wss.DefaultHiveotWssModuleID
	connectURL := fmt.Sprintf("%s://%s%s", wss.HiveotWssSchema, urlParts.Host, m.wssPath)
	// set the base parameters
	m.Init(moduleID, sink, connectURL, transports.DefaultRpcTimeout)
	return m
}

// Create a websocket module using WoT messaging format
// This uses the WoT websocket protocol message converter to convert between
// the standard RRN messages and the WoT websocket message format.
//
// Incoming messages are passed to the provided sink. The sink can be nil as long as it is
// set with SetSink() before calling start.
//
// httpServer is the http server the websocket is using
// sink is the required receiver of request, response and notification messages, nil to set later but before start.
func NewWotWssModule(httpServer httptransport.IHttpServer, sink modules.IHiveModule) *WssModule {
	httpURL := httpServer.GetConnectURL()
	urlParts, err := url.Parse(httpURL)
	if err != nil {
		panic("NewWotWssModule: Http server has invalid URL")
	}
	m := &WssModule{
		httpServer:   httpServer,
		msgConverter: wssapi.NewWotWssMsgConverter(),
		subprotocol:  wss.SubprotocolWotWSS,
		wssPath:      wss.DefaultWotWssPath,
	}
	moduleID := wss.DefaultWotWssModuleID
	connectURL := fmt.Sprintf("%s://%s%s", wss.WotWssSchema, urlParts.Host, m.wssPath)

	m.Init(moduleID, sink, connectURL, transports.DefaultRpcTimeout)
	return m
}
