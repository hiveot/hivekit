package ssescpkg

import (
	"time"

	"github.com/hiveot/hivekit/go/modules"
	factory "github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/ssesc"
	"github.com/hiveot/hivekit/go/modules/transports/ssesc/internal/server"
)

// NewSseScServer creates a hiveot SSE-SC transport.
//
// This uses the HiveOT RRN messages as the payload without conversions.
//
// httpServer is the http server the websocket is using
// respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewSseScServer(httpServer transports.IHttpServer, respTimeout time.Duration) ssesc.ISseScTransportServer {
	transport := server.NewHiveotSseServer(httpServer, respTimeout)
	return transport
}

// Create a new instance of the Hiveot SSE-SC server using the factory environment
// This loads the httpserver module
func NewSseScServerFactory(f factory.IModuleFactory) modules.IHiveModule {
	httpServer := f.GetHttpServer(true)
	timeout := f.GetEnvironment().RpcTimeout
	return NewSseScServer(httpServer, timeout)
}
