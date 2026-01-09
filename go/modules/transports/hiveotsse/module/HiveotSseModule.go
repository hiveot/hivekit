package module

import (
	"fmt"
	"net/url"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	"github.com/hiveot/hivekit/go/modules/transports/hiveotsse"
	sseapi "github.com/hiveot/hivekit/go/modules/transports/hiveotsse/api"
	"github.com/hiveot/hivekit/go/modules/transports/httptransport"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot/td"
)

// NewHiveotSseModule is a transport module for serving the wot http-basic protocol.
// This implements the ITransportModule (and IHiveModule) interface.
//
// This WoT defined protocol is build on top of HTTP and is uni-directional.
// It is only intended for consumers and not for agents using connection reversal.
// It does not support subscribing to events or observing properties.
type HiveotSseModule struct {
	transports.TransportModuleBase

	// handler to invoke when a connection is established or disappears
	connectHandler transports.ConnectionHandler

	// SSE protocol message converter
	converter transports.IMessageConverter

	// the RRN messaging receiver
	msgAPI *sseapi.HiveotSseMsgAPI

	// actual server exposing routes
	httpServer httptransport.IHttpServer

	// the linked authenticator
	// authenticator transports.IAuthenticator

	// service *service.HiveotSseService

	// The connection address for subscription and URL to connect using SSE
	// connectAddr string
	// connectURL string

	// The SSE connection path
	ssePath string
}

// AddTDForms for connecting to SSE, Subscribe, Observe, Send Requests, read and query
// using hiveot RequestMessage and ResponseMessage envelopes.
func (srv *HiveotSseModule) AddTDForms(tdi *td.TD, includeAffordances bool) {

	// TODO: add the hiveot http endpoints
	//srv.httpBasicServer.AddOps()
	// forms are handled through the http binding
	//return srv.httpBasicServer.AddTDForms(tdi, includeAffordances)
}

// HandleRequest passes the module request messages to the API handler.
// This has nothing to do with receiving requests over HTTP.
func (m *HiveotSseModule) HandleRequest(
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

// onRequestMessage receives http request messages via http
// // This forwards it to the sink to be processed.
// func (m *HiveotSseModule) onRequestMessage(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
// 	err := m.ForwardRequest(req, replyTo)
// 	return err
// }

// Set the handler for authentication connections to this transport module.
// func (srv *HiveotSseModule) SetAuthenticationHandler(h AuthenticationHandler) {}

// Set the handler for incoming connections
// func (srv *HiveotSseModule) SetConnectionHandler(h ConnectionHandler) {
// }

// Start readies the module for use.
func (m *HiveotSseModule) Start() (err error) {
	m.TransportModuleBase.Start()

	// TODO: detect if already listening
	// Add the routes used in SSE connection and subscription requests
	m.CreateRoutes(m.ssePath, m.httpServer.GetProtectedRoute())

	// The msg handler invokes the module API.
	if err == nil {
		m.msgAPI = sseapi.NewHiveotSseMsgHandler(m)
	}
	return err
}

// Stop any running actions
func (m *HiveotSseModule) Stop() {
}

// Start a new HiveOT Http/SSE server using the given http server.
// sink is the optional receiver of request, response and notification messages, nil to set later
// The optional connect handler is invoked when connections appear and disappear
func NewHiveotSseModule(
	server httptransport.IHttpServer,
	sink modules.IHiveModule,
	connectHandler transports.ConnectionHandler) *HiveotSseModule {

	ssePath := hiveotsse.DefaultHiveotSsePath

	httpAddr := server.GetConnectURL()
	urlParts, _ := url.Parse(httpAddr)

	connectURL := fmt.Sprintf("%s://%s%s", hiveotsse.HiveotSSESchema, urlParts.Host, ssePath)

	// use the RRN message format. Simple passthrough.
	converter := direct.NewPassthroughMessageConverter()

	m := &HiveotSseModule{
		httpServer:     server,
		connectHandler: connectHandler,
		ssePath:        ssePath,
		converter:      converter,
	}
	moduleID := hiveotsse.DefaultHiveotSseThingID
	m.Init(moduleID, sink, connectURL)

	// properties must match the module TM
	m.UpdateProperty(transports.PropName_NrConnections, 0)

	var _ transports.ITransportModule = m // interface check

	return m
}
