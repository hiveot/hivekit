package wsspkg

import (
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/wss"
	internal "github.com/hiveot/hivekit/go/modules/transport/wss/internal/server"
)

// NewHiveotWssServer creates a websocket transport using the HiveOT RRN messaging format.
//
// This uses the HiveOT RRN messages as the payload without conversions.
//
// httpServer is the http server the websocket is using
// respTimeout is the time the server waits for a response when sending requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewHiveotWssServer(
	httpServer transport.IHttpServer, respTimeout time.Duration) wss.IWssTransportServer {

	wssTransport := internal.NewHiveotWssServer(httpServer, respTimeout)
	return wssTransport
}

// Load the HiveOT websocket server using the factory environment
// This loads the http server
func NewHiveotWssServerFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	httpServer := f.GetHttpServer(true)
	if httpServer == nil {
		return nil, fmt.Errorf("NewHiveotWssServerFactory: missing http server")
	}
	timeout := f.GetEnvironment().RpcTimeout
	return NewHiveotWssServer(httpServer, timeout), nil
}

// NewWotServer creates a websocket module using WoT Websocket messaging format.
//
// This uses the WoT websocket protocol message converter to convert between
// the standard RRN messages and the WoT websocket message format.
//
// httpServer is the http server the websocket is using (required)
// respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewWotWssServer(
	httpServer transport.IHttpServer, respTimeout time.Duration) wss.IWssTransportServer {

	wssTransport := internal.NewWotWssServer(httpServer, respTimeout)
	return wssTransport
}

// Load the Wot websocket server using the factory environment
// This loads the http server.
// This returns nil if the http server could not be loaded.
func NewWotWssServerFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	httpServer := f.GetHttpServer(true)
	if httpServer == nil {
		return nil, fmt.Errorf("NewWotWssServerFactory: missing http server")
	}
	timeout := f.GetEnvironment().RpcTimeout
	return NewWotWssServer(httpServer, timeout), nil
}
