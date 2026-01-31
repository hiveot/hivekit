package httpbasicserver

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot/td"
)

// HTTP-basic profile constants
const (
	// static file server routes
	DefaultHttpStaticBase      = "/static"
	DefaultHttpStaticDirectory = "stores/httpstatic" // relative to home
)

// NewHttpBasicModule is a transport module for serving the wot http-basic protocol.
// This implements the ITransportModule and IHiveModule interfaces.
//
// This WoT defined protocol is build on top of HTTP and is uni-directional.
// It is only intended for consumers and not for agents using connection reversal.
// It does not support subscribing to events or observing properties.
type HttpBasicServer struct {
	transports.TransportServerBase

	// the RRN messaging receiver
	// this handles request for this module
	msgHandler *HttpBasicMsgHandler

	// actual httpServer exposing routes
	httpServer transports.IHttpServer

	// handler for received request messages
	serverRequestHandler msg.RequestHandler
}

// GetForm returns a form for the given operation
// Intended for updating TD's with forms to invoke a request
func (m *HttpBasicServer) GetForm(operation string, thingID string, name string) *td.Form {
	// TODO: use the standard path /operation/thingID/name
	return nil
}

// GetTM returns the module's TM describing its properties, actions and events.
// This server does not expose a TM.
func (m *HttpBasicServer) GetTM() string {
	return ""
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

	if req.ThingID == m.GetModuleID() {
		err = m.msgHandler.HandleRequest(req, replyTo)
	} else {
		err = fmt.Errorf("SendRequest. HTTP can't send requests to remote clients.")
		slog.Error(err.Error())
	}
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
func (m *HttpBasicServer) Start(yamlConfig string) (err error) {

	slog.Info("Starting http-basic server module")
	m.createRoutes()

	// The basic msg handler converts incoming module requests messages to the module API.
	// This has nothing to do with the http server.
	if err == nil {
		m.msgHandler = NewHttpBasicMsgHandler(m)
	}
	return err
}

// Stop any running actions
func (m *HttpBasicServer) Stop() {

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
	moduleID := httpbasic.DefaultHttpBasicThingID
	connectURL := httpServer.GetConnectURL()

	m.Init(moduleID, connectURL, transports.DefaultRpcTimeout)

	// TODO: properties must match the module TM
	// m.UpdateProperty(transports.PropName_NrConnections, 0)

	var _ transports.ITransportServer = m // interface check
	var _ modules.IHiveModule = m         // interface check

	return m
}
