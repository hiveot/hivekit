package serverimpl

import (
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/cells/transport"
	"github.com/hiveot/hivekit/go/cells/transport/ssesc"
	"github.com/teris-io/shortid"
)

// SseScServerImpl is a transport for serving the HiveOT SSE-SC transport protocol.
// This implements the ITransportServer (and IHiveCell) interface.
//
// This transport protocol is build on top of HTTP and is bi-directional.
// It supports subscribing to events or observing properties.
type SseScServerImpl struct {
	// Transport base includes the RnR channel for matching request-response messages.
	*transport.TransportServerBase

	// SSE-Sc protocol message encoder
	encoder transport.IMessageEncoder

	// actual server exposing routes
	httpServer api.IHttpServer

	// waiting for response timeout (see rnr)
	respTimeout time.Duration

	// The SSE connection path
	ssePath string

	// serverTD is the TD describing how to connect to this server
	serverTD *td.TD
}

// GetTD returns the server TD, containing connection and authentication information
func (srv *SseScServerImpl) GetTD() *td.TD {
	return srv.serverTD
}

// Start readies the server for use.
//
// yamlConfig todo configure ssepath
func (srv *SseScServerImpl) Start() (err error) {

	slog.Info("Start: Starting ssesc transport server", "ssePath", srv.ssePath)

	// Add the routes used in SSE connection and subscription requests
	srv.CreateRoutes(srv.ssePath, srv.httpServer.GetProtectedRoute())

	// create a TD describing this server along with its connection URL
	thingID := srv.GetThingID()
	srv.serverTD = td.NewTD(thingID, "SSE-SC server", vocab.DeviceTypeService)
	srv.AddTDSecForms(srv.serverTD, false)
	return err
}

// Stop any running actions
func (srv *SseScServerImpl) Stop() {
	slog.Info("Stop: Stopping ssesc transport server")
	srv.CloseAll()
}

// Start a new HiveOT Http/SSE server using the given http server.
// The http server must have authentication setup
//
// # The optional connect handler is invoked when connections appear and disappear
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by Things.
func NewSseScServerImpl(httpServer api.IHttpServer, respTimeout time.Duration) *SseScServerImpl {

	ssePath := ssesc.SseScPath

	httpAddr := httpServer.GetConnectURL()
	urlParts, _ := url.Parse(httpAddr)

	connectURL := fmt.Sprintf("%s://%s%s", api.HiveotSseScScheme, urlParts.Host, ssePath)

	// use the RRN message format. Simple passthrough.
	encoder := transport.NewRRNJsonEncoder()
	if respTimeout == 0 {
		respTimeout = msg.DefaultRnRTimeout
	}

	thingID := ssesc.SseScServerCellType + "-" + shortid.MustGenerate()
	authenticator := httpServer.GetAuthenticator()
	m := &SseScServerImpl{
		TransportServerBase: transport.NewTransportServerBase(thingID, connectURL, authenticator),
		httpServer:          httpServer,
		ssePath:             ssePath,
		encoder:             encoder,
		respTimeout:         respTimeout,
	}

	var _ api.IHiveCell = m        // interface check
	var _ api.ITransportServer = m // interface check

	return m
}
