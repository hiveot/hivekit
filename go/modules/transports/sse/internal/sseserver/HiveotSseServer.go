package sseserver

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	sseapi "github.com/hiveot/hivekit/go/modules/transports/sse/api"
	"github.com/hiveot/hivekit/go/msg"
)

// HiveotSseServer is a transport module for serving the HiveOT SSE-SC transport protocol.
// This implements the ITransportModule (and IHiveModule) interface.
//
// This transport protocol is build on top of HTTP and is bi-directional.
// It supports subscribing to events or observing properties.
type HiveotSseServer struct {
	// Transport base includes the RnR channel for matching request-response messages.
	transports.TransportServerBase

	// SSE-Sc protocol message converter
	converter transports.IMessageConverter

	// the RRN messaging receiver
	msgAPI *HiveotSseMsgHandler

	// actual server exposing routes
	httpServer transports.IHttpServer

	// The connection address for subscription and URL to connect using SSE
	// connectAddr string
	// connectURL string

	// waiting for response timeout (see rnr)
	respTimeout time.Duration

	// The SSE connection path
	ssePath string
}

// Get the agent/producer connection that serves the given ThingID
// This supports using an agent prefix separated by ':' for the thingID
func (m *HiveotSseServer) DetermineAgentConnection(thingID string) (transports.IConnection, error) {
	parts := strings.Split(thingID, ":")
	agentID := parts[0]

	c := m.GetConnectionByClientID(agentID)
	if c == nil {
		return nil, fmt.Errorf("No connection found for ThingID '%s'", thingID)
	}
	return c, nil
}

func (m *HiveotSseServer) GetProtocolType() string {
	return transports.HiveotSseScProtocolType
}

// Handle a notification this module (or downstream in the chain) subscribed to.
// Notifications are forwarded to their upstream sink, which for a server is the
// client.
func (m *HiveotSseServer) HandleNotification(notif *msg.NotificationMessage) {
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
func (m *HiveotSseServer) HandleRequest(
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
func (m *HiveotSseServer) Start(yamlConfig string) (err error) {

	// TODO: detect if already listening
	// Add the routes used in SSE connection and subscription requests
	m.CreateRoutes(m.ssePath, m.httpServer.GetProtectedRoute())

	// The msg handler invokes the module API.
	m.msgAPI = NewHiveotSseMsgHandler(m)
	return err
}

// Stop any running actions
func (m *HiveotSseServer) Stop() {
}

// Start a new HiveOT Http/SSE server using the given http server.
// The http server must have authentication setup
//
// # The optional connect handler is invoked when connections appear and disappear
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewHiveotSseServer(httpServer transports.IHttpServer, respTimeout time.Duration) *HiveotSseServer {

	ssePath := sseapi.HiveotSseScPath

	httpAddr := httpServer.GetConnectURL()
	urlParts, _ := url.Parse(httpAddr)

	connectURL := fmt.Sprintf("%s://%s%s", transports.HiveotSseScUrlScheme, urlParts.Host, ssePath)

	// use the RRN message format. Simple passthrough.
	converter := direct.NewPassthroughMessageConverter()
	if respTimeout == 0 {
		respTimeout = transports.DefaultRpcTimeout
	}

	m := &HiveotSseServer{
		httpServer:  httpServer,
		ssePath:     ssePath,
		converter:   converter,
		respTimeout: respTimeout,
	}
	moduleID := sseapi.HiveotSseScModuleID
	m.Init(moduleID, transports.HiveotSseScSubprotocol, connectURL, httpServer.GetAuthenticator())

	var _ modules.IHiveModule = m         // interface check
	var _ transports.ITransportServer = m // interface check

	return m
}
