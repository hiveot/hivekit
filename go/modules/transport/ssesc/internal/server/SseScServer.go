package internal

import (
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/ssesc"
	"github.com/teris-io/shortid"
)

// SseScServer is a transport module for serving the HiveOT SSE-SC transport protocol.
// This implements the ITransportModule (and IHiveModule) interface.
//
// This transport protocol is build on top of HTTP and is bi-directional.
// It supports subscribing to events or observing properties.
type SseScServer struct {
	// Transport base includes the RnR channel for matching request-response messages.
	*transport.TransportServerBase

	// SSE-Sc protocol message encoder
	encoder transport.IMessageEncoder

	// the RRN messaging receiver
	// msgAPI *HiveotSseScMsgHandler

	// actual server exposing routes
	httpServer transport.IHttpServer

	// The connection address for subscription and URL to connect using SSE
	// connectAddr string
	// connectURL string

	// waiting for response timeout (see rnr)
	respTimeout time.Duration

	// The SSE connection path
	ssePath string
}

func (m *SseScServer) GetProtocolType() (string, string) {
	return transport.ProtocolTypeHiveotSsesc, transport.SubprotocolHiveotSsesc
}

// Start readies the module for use.
//
// yamlConfig todo configure ssepath
func (m *SseScServer) Start() (err error) {

	slog.Info("Start: Starting ssesc transport server")

	// Add the routes used in SSE connection and subscription requests
	m.CreateRoutes(m.ssePath, m.httpServer.GetProtectedRoute())

	// The handler for messaging requests directed at this module
	// m.msgAPI = NewHiveotSseMsgHandler(m)
	return err
}

// Stop any running actions
func (m *SseScServer) Stop() {
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
func NewSseScServer(httpServer transport.IHttpServer, respTimeout time.Duration) *SseScServer {

	ssePath := ssesc.SseScPath

	httpAddr := httpServer.GetConnectURL()
	urlParts, _ := url.Parse(httpAddr)

	connectURL := fmt.Sprintf("%s://%s%s", transport.ProtocolSchemeHiveotSseSc, urlParts.Host, ssePath)

	// use the RRN message format. Simple passthrough.
	encoder := transport.NewRRNJsonEncoder()
	if respTimeout == 0 {
		respTimeout = msg.DefaultRnRTimeout
	}

	thingID := ssesc.SseScServerModuleType + "-" + shortid.MustGenerate()
	authenticator := httpServer.GetAuthenticator()
	m := &SseScServer{
		TransportServerBase: transport.NewTransportServerBase(thingID, connectURL, authenticator),
		httpServer:          httpServer,
		ssePath:             ssePath,
		encoder:             encoder,
		respTimeout:         respTimeout,
	}

	var _ modules.IHiveModule = m        // interface check
	var _ transport.ITransportServer = m // interface check

	return m
}
