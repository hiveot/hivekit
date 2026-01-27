package ssescserver

import (
	"fmt"
	"net/url"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	"github.com/hiveot/hivekit/go/modules/transports/ssesc"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot/td"
)

// SsescServer is a transport module for serving the HiveOT SSE-SC transport protocol.
// This implements the ITransportModule (and IHiveModule) interface.
//
// This transport protocol is build on top of HTTP and is bi-directional.
// It supports subscribing to events or observing properties.
type SsescServer struct {
	// Transport base includes the RnR channel for matching request-response messages.
	transports.TransportModuleBase

	// handler to invoke when a connection is established or disappears
	connectHandler transports.ConnectionHandler

	// SSE protocol message converter
	converter transports.IMessageConverter

	// the RRN messaging receiver
	msgAPI *SseScMsgHandler

	// actual server exposing routes
	httpServer transports.IHttpServer

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
func (srv *SsescServer) AddTDForms(tdi *td.TD, includeAffordances bool) {

	// TODO: add the hiveot http endpoints
	//srv.httpBasicServer.AddOps()
	// forms are handled through the http binding
	//return srv.httpBasicServer.AddTDForms(tdi, includeAffordances)
}

// HandleRequest passes the module request messages to the API handler.
// This has nothing to do with receiving requests over HTTP.
func (m *SsescServer) HandleRequest(
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

// Set the handler for authentication connections to this transport module.
// func (srv *HiveotSseModule) SetAuthenticationHandler(h AuthenticationHandler) {}

// Set the handler for incoming connections
// func (srv *HiveotSseModule) SetConnectionHandler(h ConnectionHandler) {
// }

// Start readies the module for use.
//
// yamlConfig todo configure ssepath
func (m *SsescServer) Start(yamlConfig string) (err error) {
	m.TransportModuleBase.Start("")

	// TODO: detect if already listening
	// Add the routes used in SSE connection and subscription requests
	m.CreateRoutes(m.ssePath, m.httpServer.GetProtectedRoute())

	// The msg handler invokes the module API.
	m.msgAPI = NewSseScMsgHandler(m)
	return err
}

// Stop any running actions
func (m *SsescServer) Stop() {
}

// Start a new HiveOT Http/SSE server using the given http server.
// The http server must have authentication setup
//
// sink is the optional receiver of request, response and notification messages, nil to set later
// The optional connect handler is invoked when connections appear and disappear
func NewHiveotSsescServer(
	server transports.IHttpServer,
	sink modules.IHiveModule,
	connectHandler transports.ConnectionHandler) *SsescServer {

	ssePath := ssesc.DefaultSseScPath

	httpAddr := server.GetConnectURL()
	urlParts, _ := url.Parse(httpAddr)

	connectURL := fmt.Sprintf("%s://%s%s", ssesc.HiveotSsescSchema, urlParts.Host, ssePath)

	// use the RRN message format. Simple passthrough.
	converter := direct.NewPassthroughMessageConverter()

	m := &SsescServer{
		httpServer:     server,
		connectHandler: connectHandler,
		ssePath:        ssePath,
		converter:      converter,
	}
	moduleID := ssesc.DefaultSseScThingID
	m.Init(moduleID, sink, connectURL, transports.DefaultRpcTimeout)

	// properties must match the module TM
	m.UpdateProperty(transports.PropName_NrConnections, 0)

	var _ transports.ITransportModule = m // interface check

	return m
}
