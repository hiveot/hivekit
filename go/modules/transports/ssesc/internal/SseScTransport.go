package internal

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	ssescapi "github.com/hiveot/hivekit/go/modules/transports/ssesc/api"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot/td"
)

// SsescTransport is a transport module for serving the HiveOT SSE-SC transport protocol.
// This implements the ITransportModule (and IHiveModule) interface.
//
// This transport protocol is build on top of HTTP and is bi-directional.
// It supports subscribing to events or observing properties.
type SsescTransport struct {
	// Transport base includes the RnR channel for matching request-response messages.
	transports.TransportServerBase

	// handler to invoke when a connection is established or disappears
	// connectHandler transports.ConnectionHandler

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

	// waiting for response timeout (see rnr)
	respTimeout time.Duration

	// The SSE connection path
	ssePath string
}

// AddTDForms for connecting to SSE, Subscribe, Observe, Send Requests, read and query
// using hiveot RequestMessage and ResponseMessage envelopes.
func (srv *SsescTransport) AddTDSecForms(tdoc *td.TD, includeAffordances bool) {

	// 2. Set the security scheme used by the authenticator.
	authr := srv.httpServer.GetAuthenticator()
	authr.AddSecurityScheme(tdoc)

	// TODO: add the hiveot http endpoints
	//srv.httpBasicServer.AddOps()
	// forms are handled through the http binding
	//return srv.httpBasicServer.AddTDForms(tdi, includeAffordances)
}

// Get the agent/producer connection that serves the given ThingID
// This supports using an agent prefix separated by ':' for the thingID
func (m *SsescTransport) DetermineAgentConnection(thingID string) (transports.IConnection, error) {
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
func (m *SsescTransport) HandleNotification(notif *msg.NotificationMessage) {
	m.SendNotification(notif)
}

// HandleRequest handles requests directed at this module or a connected agent.
// If not directed to this module then forward the request to the remote client.
// This means that a consumer running on the server sends a request to a producer
// connected as a client using connection reversal.
// The ThingID in the request must match the clientID of a connected client.
//
// This returns an error when the destination for the request cannot be found.
// If multiple server protocols are used it is okay to try them one by one.
func (m *SsescTransport) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// first attempt to procss the when targeted at this module
	if req.ThingID == m.GetModuleID() {
		err = m.msgAPI.HandleRequest(req, replyTo)
	} else {
		// if the request is not for this server, then send the request to the connected agent
		var c transports.IConnection
		// if the request is not for this module then pass it to the remote connection
		c, err := m.DetermineAgentConnection(req.ThingID)
		if err == nil {
			err = c.SendRequest(req, replyTo)
		}
	}
	return err
}

// Start readies the module for use.
//
// yamlConfig todo configure ssepath
func (m *SsescTransport) Start(yamlConfig string) (err error) {

	// TODO: detect if already listening
	// Add the routes used in SSE connection and subscription requests
	m.CreateRoutes(m.ssePath, m.httpServer.GetProtectedRoute())

	// The msg handler invokes the module API.
	m.msgAPI = NewSseScMsgHandler(m)
	return err
}

// Stop any running actions
func (m *SsescTransport) Stop() {
}

// Start a new HiveOT Http/SSE server using the given http server.
// The http server must have authentication setup
//
// # The optional connect handler is invoked when connections appear and disappear
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewSseScTransport(server transports.IHttpServer, respTimeout time.Duration) *SsescTransport {

	ssePath := ssescapi.DefaultSseScPath

	httpAddr := server.GetConnectURL()
	urlParts, _ := url.Parse(httpAddr)

	connectURL := fmt.Sprintf("%s://%s%s", ssescapi.HiveotSsescSchema, urlParts.Host, ssePath)

	// use the RRN message format. Simple passthrough.
	converter := direct.NewPassthroughMessageConverter()
	if respTimeout == 0 {
		respTimeout = transports.DefaultRpcTimeout
	}

	m := &SsescTransport{
		httpServer:  server,
		ssePath:     ssePath,
		converter:   converter,
		respTimeout: respTimeout,
	}
	moduleID := ssescapi.DefaultSseScThingID
	m.Init(moduleID, connectURL)

	var _ modules.IHiveModule = m         // interface check
	var _ transports.ITransportServer = m // interface check

	return m
}
