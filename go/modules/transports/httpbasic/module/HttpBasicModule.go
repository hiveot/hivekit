package module

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic/httpbasicapi"
	"github.com/hiveot/hivekit/go/modules/transports/httptransport"
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
// This implements the ITransportModule (and IHiveModule) interface.
//
// This WoT defined protocol is build on top of HTTP and is uni-directional.
// It is only intended for consumers and not for agents using connection reversal.
// It does not support subscribing to events or observing properties.
type HttpBasicModule struct {
	transports.TransportModuleBase

	// the RRN messaging receiver
	// this handles request for this module
	msgAPI *httpbasicapi.HttpBasicMsgAPI

	// actual httpServer exposing routes
	httpServer httptransport.IHttpServer

	// handler for received request messages
	serverRequestHandler msg.RequestHandler
}

// GetForm returns a form for the given operation
// Intended for updating TD's with forms to invoke a request
func (m *HttpBasicModule) GetForm(operation string, thingID string, name string) *td.Form {
	// TODO: use the standard path /operation/thingID/name
	return nil
}

// HandleRequest passes the module request messages to the API handler.
// This has nothing to do with receiving requests over HTTP.
func (m *HttpBasicModule) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

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
func (m *HttpBasicModule) Start() (err error) {
	err = m.TransportModuleBase.Start()

	slog.Info("Starting http-basic server module")
	m.createRoutes()

	// The basic msg handler converts incoming module requests messages to the module API.
	// This has nothing to do with the http server.
	if err == nil {
		m.msgAPI = httpbasicapi.NewHttpBasicMsgAPI(m)
	}
	return err
}

// Stop any running actions
func (m *HttpBasicModule) Stop() {

}

// NewHttpBasicModule creates a new WoT http-basic protocol binding.
//
// Intended as a last-resort server as this only handles consumer connections and
// does not support subscription.
// The onRequest handler only handles responses that are sent via replyTo in a short
// timeframe. (eg timeout setting)
//
//	httpServer is the http server that listens for messages
//	sink is the optional receiver of request, response and notification messages, nil to set later
func NewHttpBasicModule(httpServer httptransport.IHttpServer,
	sink modules.IHiveModule) *HttpBasicModule {

	m := &HttpBasicModule{
		httpServer: httpServer,
	}
	moduleID := httpbasic.DefaultHttpBasicThingID
	connectURL := httpServer.GetConnectURL()

	m.Init(moduleID, sink, connectURL, transports.DefaultRpcTimeout)

	// properties must match the module TM
	m.UpdateProperty(transports.PropName_NrConnections, 0)

	var _ transports.ITransportModule = m // interface check

	return m
}
