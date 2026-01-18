package ssesc

import "github.com/hiveot/hivekit/go/modules/transports"

// SSEPingEvent used by the server to ping the client that the connection is ready
const SSEPingEvent = "sse-ping"

// DefaultSseScThingID is the default thingID of the sse-sc transport module.
const DefaultSseScThingID = "hiveot-ssesc"

const (
	DefaultSseScPath = "/hiveot/ssesc"

	// PostSseScRequestPath HTTP endpoint that accepts HiveOT RequestMessage envelopes
	PostSseScRequestPath = "/hiveot/request"

	// PostSseScResponsePath HTTP endpoint that accepts HiveOT ResponseMessage envelopes
	PostSseScResponsePath = "/hiveot/response"

	// PostHiveotSseNotificationPath HTTP endpoint that accepts HiveOT NotificationMessage envelopes
	PostSseScNotificationPath = "/hiveot/notification"

	SseScOpConnect    = "ssesc-connect"
	HiveotSseScSchema = "sse-sc"
)

// Interface of the HiveotSse module services
type ISseScTransport interface {
	transports.ITransportModule
	// todo: future API  for servicing the module
}
