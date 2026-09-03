package ssesc_server

import (
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/transport/ssesc"
	"github.com/hiveot/hivekit/go/cells/transport/ssesc/internal/serverimpl"
)

// StartSseScServer creates a hiveot SSE-SC transport.
//
// This uses the HiveOT RRN messages as the payload without conversions.
//
// httpServer is the http server the websocket is using
// respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by devices.
func StartSseScServer(
	httpServer api.IHttpServer, respTimeout time.Duration) (ssesc.ISseScTransportServer, error) {

	return serverimpl.StartSseScServerImpl(httpServer, respTimeout)
}

// Create a new instance of the Hiveot SSE-SC server using the factory environment
// This loads the httpserver cell.
func StartSseScServerFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	httpServer := f.GetHttpServer(true)
	if httpServer == nil {
		return nil, fmt.Errorf("NewSseScServerFactory: missing http server")
	}
	timeout := f.GetEnvironment().RpcTimeout
	return StartSseScServer(httpServer, timeout)
}
