package ssescpkg

import (
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/transport/ssesc"
	internal "github.com/hiveot/hivekit/go/modules/transport/ssesc/internal/server"
)

// NewSseScServer creates a hiveot SSE-SC transport.
//
// This uses the HiveOT RRN messages as the payload without conversions.
//
// httpServer is the http server the websocket is using
// respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by devices.
func NewSseScServer(httpServer api.IHttpServer, respTimeout time.Duration) ssesc.ISseScTransportServer {
	transport := internal.NewSseScServer(httpServer, respTimeout)
	return transport
}

// Create a new instance of the Hiveot SSE-SC server using the factory environment
// This loads the httpserver module
func NewSseScServerFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	httpServer := f.GetHttpServer(true)
	if httpServer == nil {
		return nil, fmt.Errorf("NewSseScServerFactory: missing http server")
	}
	timeout := f.GetEnvironment().RpcTimeout
	return NewSseScServer(httpServer, timeout), nil
}
