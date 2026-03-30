package ssetransport

import (
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	sseapi "github.com/hiveot/hivekit/go/modules/transports/sse/api"
	"github.com/hiveot/hivekit/go/modules/transports/sse/internal/sseserver"
)

// NewHiveotSseServer creates a hiveot SSE-SC transport.
//
// This uses the HiveOT RRN messages as the payload without conversions.
//
// httpServer is the http server the websocket is using
// respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewHiveotSseServer(httpServer transports.IHttpServer, respTimeout time.Duration) sseapi.ISseTransportServer {
	transport := sseserver.NewHiveotSseServer(httpServer, respTimeout)
	return transport
}

// NewWotSseServer creates a WoT SSE server transport.
//
// httpServer is the http server the websocket is using
// respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
// func NewWotSseServer(httpServer transports.IHttpServer, respTimeout time.Duration) sseapi.ISseTransportServer {

// 	transport := sseserver.NewWotSseServer(httpServer, respTimeout)
// 	return transport
// }
