package ssesc

import (
	"time"

	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	ssescapi "github.com/hiveot/hivekit/go/modules/transports/ssesc/api"
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
func NewSseScServer(httpServer transports.IHttpServer, respTimeout time.Duration) ssescapi.ISseScTransportServer {
	transport := server.NewHiveotSseServer(httpServer, respTimeout)
	return transport
}

// Create a new instance of the Hiveot SSE-SC server using the factory environment
// This loads the httpserver module
func NewSseScServerFactory(f factoryapi.IModuleFactory) modules.IHiveModule {
	httpServer := f.GetHttpServer()
	timeout := f.GetEnvironment().RpcTimeout
	return NewSseScServer(httpServer, timeout)
}
