package hiveotsse

import "github.com/hiveot/hivekit/go/modules/transports"

// DefaultHiveotSseThingID is the default thingID of the transport module.
const DefaultHiveotSseThingID = "hiveot-sse"

const (
	DefaultHiveotSsePath = "/hiveot/sse"

	// PostHiveotSseRequestPath HTTP endpoint that accepts HiveOT RequestMessage envelopes
	PostHiveotSseRequestPath = "/hiveot/request"

	// PostHiveotSseResponsePath HTTP endpoint that accepts HiveOT ResponseMessage envelopes
	PostHiveotSseResponsePath = "/hiveot/response"

	// PostHiveotSseNotificationPath HTTP endpoint that accepts HiveOT NotificationMessage envelopes
	PostHiveotSseNotificationPath = "/hiveot/notification"

	SSEOpConnect    = "sse-connect"
	HiveotSSESchema = "sse"
)

// Interface of the HiveotSse module services
type IHiveotSseTransport interface {
	transports.ITransportModule
	// todo: future API  for servicing the module
}
