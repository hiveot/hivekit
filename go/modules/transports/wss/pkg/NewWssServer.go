package wsspkg

import (
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/wss"
	internal "github.com/hiveot/hivekit/go/modules/transports/wss/internal/server"
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
	httpServer transports.IHttpServer, respTimeout time.Duration) wss.IWssTransportServer {

	wssTransport := internal.NewHiveotWssServer(httpServer, respTimeout)
	return wssTransport
}

// NewWotServer creates a websocket module using WoT Websocket messaging format.
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
	httpServer transports.IHttpServer, respTimeout time.Duration) wss.IWssTransportServer {

	wssTransport := internal.NewWotWssServer(httpServer, respTimeout)
	return wssTransport
}
