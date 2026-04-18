package wsspkg

import (
	"time"

	"github.com/hiveot/hivekit/go/modules"
	factory "github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
	wssapi "github.com/hiveot/hivekit/go/modules/transports/wss"
	"github.com/hiveot/hivekit/go/modules/transports/wss/internal/server"
)

// NewHiveotWssServer creates a websocket transport using the HiveOT RRN messaging format.
//
// This uses the HiveOT RRN messages as the payload without conversions.
//
// httpServer is the http server the websocket is using
// respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewHiveotWssServer(
	httpServer transports.IHttpServer, respTimeout time.Duration) wssapi.IWssTransportServer {

	wssTransport := server.NewHiveotWssServer(httpServer, respTimeout)
	return wssTransport
}

// Load the HiveOT websocket server using the factory environment
// This loads the http server
func NewHiveotWssServerFactory(f factory.IModuleFactory) modules.IHiveModule {
	httpServer := f.GetHttpServer()
	timeout := f.GetEnvironment().RpcTimeout
	return NewHiveotWssServer(httpServer, timeout)
}

// NewWotWssServer creates a websocket module using WoT Websocket messaging format.
//
// This uses the WoT websocket protocol message converter to convert between
// the standard RRN messages and the WoT websocket message format.
//
// httpServer is the http server the websocket is using
// respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewWotWssServer(
	httpServer transports.IHttpServer, respTimeout time.Duration) wssapi.IWssTransportServer {

	wssTransport := server.NewWotWssServer(httpServer, respTimeout)
	return wssTransport
}

// Load the Wot websocket server using the factory environment
// This loads the http server
func NewWotWssServerFactory(f factory.IModuleFactory) modules.IHiveModule {
	httpServer := f.GetHttpServer()
	timeout := f.GetEnvironment().RpcTimeout
	return NewWotWssServer(httpServer, timeout)
}
