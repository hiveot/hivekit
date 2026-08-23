package ssesc

import (
	"github.com/hiveot/hivekit/go/api"
)

// SSEPingEvent used by the server to ping the client that the connection is ready
const SSEPingEvent = "sse-ping"

const (
	// Hide type of the hiveot HTTP/SSE-SC server
	SseScClientCellType = "hiveot-ssesc-client"
	SseScServerCellType = "hiveot-ssesc-server"

	SseScPath = "/hiveot/ssesc"

	// Well-known hiveot request endpoint carrying a RequestMessage envelope
	PostSseScRequestPath = "/hiveot/request"

	// Well-known hiveot response endpoint carrying a ResponseMessage envelope
	PostSseScResponsePath = "/hiveot/response"

	// Well-known hiveot notification endpoint carrying a NotificationMessage envelope
	PostSseScNotificationPath = "/hiveot/notification"

	SseScOpConnect = "ssesc-connect"
)

// Interface of the HiveotSseSc transport server
type ISseScTransportServer interface {
	api.ITransportServer
}
