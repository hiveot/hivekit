package server

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	httpbasicapi "github.com/hiveot/hivekit/go/modules/transports/httpbasic/api"
)

// HTTP-basic profile constants
const (
	// static file server routes
	DefaultHttpStaticBase      = "/static"
	DefaultHttpStaticDirectory = "stores/httpstatic" // relative to home
)

// HttpBasicServer is a transport module for serving the wot http-basic protocol.
// This implements the ITransportModule and IHiveModule interfaces.
//
// This WoT defined protocol is build on top of HTTP and is uni-directional.
// It is only intended for consumers and not for agents using connection reversal.
// It does not support subscribing to events or observing properties.
type HttpBasicServer struct {
	transports.TransportServerBase

	// actual httpServer exposing routes
	httpServer transports.IHttpServer

	// reqHandler handles the requests received from the remote consumer
	// requestHandler msg.RequestHandler
}

// Handle a notification this module (or downstream in the chain) subscribed to.
// Notifications are forwarded to their upstream sink, which for a server is the
// client.
func (m *HttpBasicServer) HandleNotification(notif *msg.NotificationMessage) {
	m.SendNotification(notif)
}

// HandleRequest passes the module request messages to the API handler.
// If the request isn't for this module then this returns an error as the server
// cannot deliver messages to the client.
func (m *HttpBasicServer) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	err = fmt.Errorf("SendRequest. HTTP can't send requests to remote clients.")
	slog.Error(err.Error())
	return err
}

// Start readies the module for use.
//
// Configurable:
// - add public routes for ping
// - add protected route for thing requests {operation}/{thing}/{name}
// - add protected route for affordance requests {operation}/{thing}/{affordance}/{name}
//
// Incoming requests are converted to the standard message format and passed to
// the registered sinks.
//
// This supports the HandleRequest - read(all)properties RRN to retrieve statistics
// of the http transport.
//
// Since http is a unidirectional protocol, HandleNotification and HandleRequest messages
// will not be passed to connected clients.
//
// yamlConfig tbd: use base path?
func (m *HttpBasicServer) Start() (err error) {

	slog.Info("Start: Starting httpbasic transport server")
	m.createRoutes()

	return err
}

// Stop any running actions
func (m *HttpBasicServer) Stop() {
	slog.Info("Stop: Stopping httpbasic transport server")

}

// NewHttpBasicServer creates a new WoT http-basic protocol binding.
//
// Intended as a last-resort server as this only handles consumer connections and
// does not support subscription.
// The onRequest handler only handles responses that are sent via replyTo in a short
// timeframe. (eg timeout setting)
//
//	httpServer is the http server that listens for messages
//	sink is the optional receiver of request, response and notification messages, nil to set later
func NewHttpBasicServer(httpServer transports.IHttpServer) *HttpBasicServer {

	m := &HttpBasicServer{
		httpServer: httpServer,
	}

	connectURL := httpServer.GetConnectURL()
	authenticator := httpServer.GetAuthenticator()
	m.Init(
		httpbasicapi.HttpBasicServerModuleType,
		transports.ProtocolTypeWotHttpBasic,
		transports.SubprotocolWotHttpBasic,
		connectURL, authenticator)

	// TODO: properties must match the module TM
	// m.UpdateProperty(transports.PropName_NrConnections, 0)

	var _ transports.ITransportServer = m // interface check
	var _ modules.IHiveModule = m         // interface check

	return m
}
