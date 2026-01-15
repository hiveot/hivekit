package module

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/utils"
)

// WssServer is a websocket transport protocol server for use with HiveOT and WoT
// messages.
//
// Use AddEndpoint to add a service endpoint to listen on and a corresponding message converter.
//
// While intended for the Hub, it can also be used in stand-alone Things that
// run their own servers. An https server is required.
//
// The difference with the WoT Websocket protocol is that it transport the Request
// and Response messages directly as-is, using JSON encoding.
//
// Connections support event subscription and property observe requests, and sends
// updates as Responses with the subscription correlationID.
type WssServer struct {

	// manage the incoming connections
	// cm *connections.ConnectionManager

	// the connection URL for this websocket server
	// connectURL string

	// The router to register with
	// router chi.Router

	// registered handler to notify of incoming connections
	// this handler will register RRN callbacks on the connection.
	// serverConnectHandler transports.ConnectionHandler

	// registered handler of incoming notifications
	// serverNotificationHandler transports.NotificationHandler
	// registered handler of incoming requests (which return a reply)
	// serverRequestHandler transports.RequestHandler
	// registered handler of incoming responses (which sends a reply to the request sender)
	// serverResponseHandler transports.ResponseHandler

	// Conversion between websocket messages and the standard hiveot message envelope.
	// messageConverter transports.IMessageConverter

	// mutex for updating cm
	// mux sync.RWMutex

	// listening path for incoming connections
	// wssPath string
}

func (srv *WssServer) CloseAll() {
	// connection handler must close the connections
	// srv.cm.CloseAll()
}

// CloseAllClientConnections close all cm from the given client.
// Intended to close cm after a logout.
// func (srv *WssServer) CloseAllClientConnections(clientID string) {
// 	srv.cm.ForEachConnection(func(c transports.IServerConnection) {
// 		cinfo := c.GetConnectionInfo()
// 		if cinfo.ClientID == clientID {
// 			c.Disconnect()
// 		}
// 	})
// }

// GetConnectionByConnectionID returns the connection with the given connection ID
// func (srv *WssServer) GetConnectionByConnectionID(clientID, cid string) transports.IConnection {
// 	return srv.cm.GetConnectionByConnectionID(clientID, cid)
// }

// GetConnectionByClientID returns the connection with the given client ID
// func (srv *WssServer) GetConnectionByClientID(agentID string) transports.IConnection {
// 	return srv.cm.GetConnectionByClientID(agentID)
// }

// GetProtocolType returns the protocol type of this server
func (srv *WssServer) GetProtocolType() string {
	return transports.ProtocolTypeWotWSS
}

// SendNotification sends a property update or event response message to subscribers
// func (srv *WssServer) SendNotification(msg *msg.NotificationMessage) {
// 	// pass the response to all subscribed cm
// 	srv.cm.ForEachConnection(func(c transports.IServerConnection) {
// 		_ = c.SendNotification(msg)
// 	})
// }

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

// // Start listening for incoming SSE connections
// func (srv *WssServer) Start() error {
// 	slog.Info("Starting websocket server, Listening on: " + srv.GetConnectURL())
// 	// TODO: detect if already listening
// 	srv.router.Get(srv.wssPath, srv.Serve)
// 	return nil
// }

// // Stop disconnects clients and remove connection listening
// func (srv *WssServer) Stop() {
// 	srv.CloseAll()
// 	srv.router.Delete(srv.wssPath, srv.Serve)
// }

// NewWssServer returns a new websocket protocol server. Use Start() to activate routes.
//
// The user must handle connect/disconnect events and handle messages from established
// connections. See also the ConnectionManager module that can perform this task.
//
// connectAddr is the host:port of the webserver
// wsspath is the path of the websocket endpoint that will listen on the server
// router is the protected route that serves websocket on the wssPath
// handleConnect connection handling callback to listen for connect/disconnect notifications
// func NewWssServer(
// 	connectAddr string,
// 	wssPath string,
// 	router chi.Router,
// 	handleConnect transports.ConnectionHandler,
// 	// handleNotification transports.NotificationHandler,
// 	// handleRequest transports.RequestHandler,
// 	// handleResponse transports.ResponseHandler,
// ) *WssServer {

// 	connectURL := fmt.Sprintf("%s://%s%s", wotwss.WssSchema, connectAddr, wssPath)
// 	parts, _ := url.Parse(wssURL)
// 	if parts.Port() == "" {
// 		parts.Host = fmt.Sprintf("%s:%d", parts.Hostname(), servers.DefaultHttpsPort)
// 		wssURL = parts.String()
// 	}

// 	srv := &WssServer{
// 		connectURL:           connectURL,
// 		serverConnectHandler: handleConnect,
// 		// serverNotificationHandler: handleNotification,
// 		// serverRequestHandler:      handleRequest,
// 		// serverResponseHandler:     handleResponse,
// 		messageConverter: wotwssapi.NewWotWssMsgConverter(),
// 		// cm:                        connections.NewConnectionManager(),
// 		router:  router,
// 		wssPath: wssPath,
// 	}
// 	return srv
// }
