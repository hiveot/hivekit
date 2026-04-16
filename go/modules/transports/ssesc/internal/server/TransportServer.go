package server

import (
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	ssescapi "github.com/hiveot/hivekit/go/modules/transports/ssesc/api"
)

// TransportServer is a transport module for serving the HiveOT SSE-SC transport protocol.
// This implements the ITransportModule (and IHiveModule) interface.
//
// This transport protocol is build on top of HTTP and is bi-directional.
// It supports subscribing to events or observing properties.
type TransportServer struct {
	// Transport base includes the RnR channel for matching request-response messages.
	transports.TransportServerBase

	// SSE-Sc protocol message encoder
	encoder transports.IMessageEncoder

	// the RRN messaging receiver
	// msgAPI *HiveotSseScMsgHandler

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

func (m *TransportServer) GetProtocolType() (string, string) {
	return transports.ProtocolTypeHiveotSsesc, transports.SubprotocolHiveotSsesc
}

// Start readies the module for use.
//
// yamlConfig todo configure ssepath
func (m *TransportServer) Start() (err error) {

	slog.Info("Start: Starting ssesc transport server")

	// Add the routes used in SSE connection and subscription requests
	m.CreateRoutes(m.ssePath, m.httpServer.GetProtectedRoute())

	// The handler for messaging requests directed at this module
	// m.msgAPI = NewHiveotSseMsgHandler(m)
	return err
}

// Stop any running actions
func (m *TransportServer) Stop() {
	slog.Info("Stop: Stopping ssesc transport server")
	m.CloseAll()
}

// Start a new HiveOT Http/SSE server using the given http server.
// The http server must have authentication setup
//
// # The optional connect handler is invoked when connections appear and disappear
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewHiveotSseServer(httpServer transports.IHttpServer, respTimeout time.Duration) *TransportServer {

	ssePath := ssescapi.SseScPath

	httpAddr := httpServer.GetConnectURL()
	urlParts, _ := url.Parse(httpAddr)

	connectURL := fmt.Sprintf("%s://%s%s", transports.UriSchemeHiveotSseSc, urlParts.Host, ssePath)

	// use the RRN message format. Simple passthrough.
	encoder := transports.NewRRNJsonEncoder()
	if respTimeout == 0 {
		respTimeout = transports.DefaultRpcTimeout
	}

	m := &TransportServer{
		httpServer:  httpServer,
		ssePath:     ssePath,
		encoder:     encoder,
		respTimeout: respTimeout,
	}
	m.Init(
		ssescapi.SseScServerModuleType,
		transports.ProtocolTypeHiveotSsesc,
		transports.SubprotocolHiveotSsesc,
		connectURL, httpServer.GetAuthenticator())

	var _ modules.IHiveModule = m         // interface check
	var _ transports.ITransportServer = m // interface check

	return m
}
