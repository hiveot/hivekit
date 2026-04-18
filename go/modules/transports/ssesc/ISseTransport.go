package ssesc

import "github.com/hiveot/hivekit/go/modules/transports"

// SSEPingEvent used by the server to ping the client that the connection is ready
const SSEPingEvent = "sse-ping"

const (
	// Hide type of the hiveot HTTP/SSE-SC server
	SseScServerModuleType = "hiveot-ssesc"

	SseScPath = "/hiveot/ssesc"

	// PostSseScRequestPath HTTP endpoint that accepts HiveOT RequestMessage envelopes
	PostSseScRequestPath = "/hiveot/request"

	// PostSseScResponsePath HTTP endpoint that accepts HiveOT ResponseMessage envelopes
	PostSseScResponsePath = "/hiveot/response"

	// PostHiveotSseNotificationPath HTTP endpoint that accepts HiveOT NotificationMessage envelopes
	PostSseScNotificationPath = "/hiveot/notification"

	SseScOpConnect = "ssesc-connect"
)

// Interface of the HiveotSseSc transport module
type ISseScTransportServer interface {
	transports.ITransportServer

	// todo: future API for configuration of the module
}
