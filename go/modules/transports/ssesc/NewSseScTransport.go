package ssesc

import (
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/ssesc/internal"
)

// NewHiveotWssTransport creates a websocket transport using the HiveOT RRN messaging format.
//
// This uses the HiveOT RRN messages as the payload without conversions.
//
// httpServer is the http server the websocket is using
// respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewSseScTransport(httpServer transports.IHttpServer, respTimeout time.Duration) transports.ITransportServer {
	transport := internal.NewSseScTransport(httpServer, respTimeout)
	return transport
}
