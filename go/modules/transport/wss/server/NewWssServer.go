package wss_server

import (
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/transport/wss"
	"github.com/hiveot/hivekit/go/modules/transport/wss/internal/serverimpl"
)

// NewHiveotWssServer creates a websocket transport using the HiveOT RRN messaging format.
//
// This uses the HiveOT RRN messages as the payload without conversions.
//
// httpServer is the http server the websocket is using
// respTimeout is the time the server waits for a response when sending requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers.
// Use SetNotificationSink to set the handler for notifications send by devices and services.
func NewHiveotWssServer(
	httpServer api.IHttpServer, respTimeout time.Duration) wss.IWssTransportServer {

	wssTransport := serverimpl.NewHiveotWssServerImpl(httpServer, respTimeout)
	return wssTransport
}

// Load the HiveOT websocket server using the factory environment
// This loads the http server
func NewHiveotWssServerFactory(
	f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {

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
// Use SetNotificationSink to set the handler for notifications send by device.
func NewWotWssServer(
	httpServer api.IHttpServer, respTimeout time.Duration) wss.IWssTransportServer {

	wssTransport := serverimpl.NewWotWssServerImpl(httpServer, respTimeout)
	return wssTransport
}

// Load the Wot websocket server using the factory environment
// This loads the http server.
// This returns nil if the http server could not be loaded.
func NewWotWssServerFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	httpServer := f.GetHttpServer(true)
	if httpServer == nil {
		return nil, fmt.Errorf("NewWotWssServerFactory: missing http server")
	}
	timeout := f.GetEnvironment().RpcTimeout
	return NewWotWssServer(httpServer, timeout), nil
}
