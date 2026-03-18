package internal

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	wssapi "github.com/hiveot/hivekit/go/modules/transports/wss/api"
	wssconverter "github.com/hiveot/hivekit/go/modules/transports/wss/converter"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
)

// WssTransport is a transport module that serves Websocket connections over http.
// This implements both ITransportServer and IHiveModule interfaces.
type WssTransport struct {
	transports.TransportServerBase

	// this handles request for this module
	msgAPI *WssRrnHandler

	// actual server exposing routes including websocket endpoint
	httpServer transports.IHttpServer

	// Websocket protocol message converter
	msgConverter transports.IMessageConverter // WoT or Hiveot message format

	// the time to wait for responses to request
	respTimeout time.Duration

	//
	subprotocol string // WoT or Hiveot subprotocol
	// listening path for incoming connections
	wssPath string
}

// // GetProtocolType returns the protocol type of this server
// func (srv *WssModule) GetProtocolType() string {
// 	return transports.ProtocolTypeWotWSS
// }

// Get the agent/producer connection that serves the given ThingID
// This supports using an agent prefix separated by ':' for the thingID
func (m *WssTransport) DetermineAgentConnection(thingID string) (transports.IConnection, error) {
	parts := strings.Split(thingID, ":")
	agentID := parts[0]

	c := m.GetConnectionByClientID(agentID)
	if c == nil {
		return nil, fmt.Errorf("No connection found for ThingID '%s'", thingID)
	}
	return c, nil
}

// Handle a notification this module (or downstream in the chain) subscribed to.
// Notifications are forwarded to their upstream sink, which for a server is the
// client.
func (m *WssTransport) HandleNotification(notif *msg.NotificationMessage) {
	m.SendNotification(notif)
}

// HandleRequest handles requests directed at this module or a connected agent.
//
// If not directed to this module then forward the request to the remote client.
// This means that a consumer running on the server sends a request to a producer
// connected as a client using connection reversal.
// The ThingID in the request must match the clientID of a connected client.
//
// This returns an error when the destination for the request cannot be found.
// If multiple server protocols are used it is okay to try them one by one.
func (m *WssTransport) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// first attempt to procss the when targeted at this module
	if req.ThingID == m.GetModuleID() {
		err = m.msgAPI.HandleRequest(req, replyTo)
	} else {
		var c transports.IConnection
		// if the request is not for this module then pass it to the remote connection
		c, err := m.DetermineAgentConnection(req.ThingID)
		if err == nil {
			err = c.SendRequest(req, replyTo)
		}
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
func (m *WssTransport) Serve(w http.ResponseWriter, r *http.Request) {
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
	c := NewWSSServerConnection(clientID, r, wssConn, m.msgConverter,
		m.ForwardRequest, m.ForwardNotification, m.respTimeout)
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
func (m *WssTransport) Start(yamlConfig string) (err error) {

	slog.Info("Start: Starting websocket module, Listening on: " + m.GetConnectURL())

	// TODO: detect if already running

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
func (m *WssTransport) Stop() {
	slog.Info("Stop: Stopping websocket module")
	m.CloseAll()
	router := m.httpServer.GetProtectedRoute()
	router.Delete(m.wssPath, m.Serve)
}

// Create a websocket server module using serving connections from consumers and agents.
//
// httpServer is the http server the websocket is using
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewHiveotWssTransport(httpServer transports.IHttpServer, respTimeout time.Duration) *WssTransport {

	httpURL := httpServer.GetConnectURL()
	urlParts, err := url.Parse(httpURL)
	if err != nil {
		panic("NewHiveotWssModule: Http server has invalid URL")
	}

	if respTimeout == 0 {
		respTimeout = transports.DefaultRpcTimeout
	}
	m := &WssTransport{
		httpServer:   httpServer,
		msgConverter: direct.NewPassthroughMessageConverter(),
		subprotocol:  wssapi.SubprotocolHiveotWSS,
		// connectHandler: nil,
		respTimeout: respTimeout,
		wssPath:     wssapi.DefaultHiveotWssPath,
	}
	moduleID := wssapi.DefaultHiveotWssModuleID
	connectURL := fmt.Sprintf("%s://%s%s", wssapi.HiveotWssSchema, urlParts.Host, m.wssPath)
	// set the base parameters
	m.Init(moduleID, connectURL)
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
func NewWotWssTransport(httpServer transports.IHttpServer, respTimeout time.Duration) *WssTransport {
	httpURL := httpServer.GetConnectURL()
	urlParts, err := url.Parse(httpURL)
	if err != nil {
		panic("NewWotWssModule: Http server has invalid URL")
	}
	if respTimeout == 0 {
		respTimeout = transports.DefaultRpcTimeout
	}
	m := &WssTransport{
		httpServer:   httpServer,
		msgConverter: wssconverter.NewWotWssMsgConverter(),
		respTimeout:  respTimeout,
		subprotocol:  wssapi.SubprotocolWotWSS,
		wssPath:      wssapi.DefaultWotWssPath,
	}
	moduleID := wssapi.DefaultWotWssModuleID
	connectURL := fmt.Sprintf("%s://%s%s", wssapi.WotWssSchema, urlParts.Host, m.wssPath)

	m.Init(moduleID, connectURL)
	// m.UpdateProperty(transports.PropName_NrConnections, 0)

	var _ modules.IHiveModule = m         // interface check
	var _ transports.ITransportServer = m // interface check

	return m
}
