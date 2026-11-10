package ssesc_server

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/transport/ssesc"
	"github.com/hiveot/hivekit/go/cells/transport/ssesc/server/internal"
)

// StartSseScServer creates a hiveot SSE-SC transport.
//
// This uses the HiveOT RRN messages as the payload without conversions.
//
// httpServer is the http server the websocket is using
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by devices.
func StartSseScServer(
	httpServer api.IHttpServer) (ssesc.ISseScTransportServer, error) {

	return internal.StartSseScServerImpl(httpServer)
}

// Create a new instance of the Hiveot SSE-SC server using the factory environment
// This loads the httpserver cell.
func StartSseScServerFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	httpServer := f.GetHttpServer(true)
	if httpServer == nil {
		return nil, fmt.Errorf("NewSseScServerFactory: missing http server")
	}
	return StartSseScServer(httpServer)
}
