package modules

import (
	"github.com/hiveot/hivekit/go/api/msg"
)

// The HiveOT module interface
// Anything that accepts requests can be a module, including clients and servers.
// This interface is the most basic module interface.
type IHiveModule interface {

	// GetModuleType returns module's Type.
	// This is not the instance ID.
	// GetModuleType() string

	// GetThingID returns the module's instance ID.
	// This is used as the sender ThingID when sending notifications.
	GetThingID() string

	// HandleRequest - invoked by consumer to this producer.
	//  [producer] processes or forwards a request downstream to other producers.
	//
	// When the request is for this module then the module processes the request and
	// invokes replyTo with the response. ReplyTo is invoked asynchronously before
	// or after returning.
	//
	// When the request is not for this producer then it is forwarded:
	//
	// 1. By default modules forward unhandled requests to their request sink.
	//    Flow: consumer -> module -[rsink]-> producer
	//
	// 2. If the module is a transport client: the request is transported to the server,
	//    and the server passes it to the producer that is registered as its sink.
	//    Flow: consumer -[rsink]-> tp-client -> tp-server -[rsink]-> producer
	//
	// 3. If the module is a transport server or server connection then the request is
	//    transported to the remote client. The client passes it to its registered sink.
	//    This sink should be a producer that can handle the request.
	//    (In this case the consumer is a process running on the server)
	//    Flow: consumer -[rsink]-> tp-server -> tp-client -[rsink]-> producer
	//
	//    Note this is the use-case where a device uses connection reversal to connect
	//         to a server, like a hub or gateway, to serve IoT data. The gateway acts
	//         as a consumer to the producer connected to the client.
	//
	//
	// A middleware module can intercept the response by forwarding the request downstream
	// while providing its own handler as the replyTo. This handler then forwards the response
	// to the original replyTo endpoint.
	//
	// This returns an error if the provided replyTo will not be able to receive a response.
	HandleRequest(request *msg.RequestMessage, replyTo msg.ResponseHandler) error

	// Handle the notification received from a producer.
	// The default behavior is to forward it upstream to the handler set with SetNotificationSink.
	HandleNotification(notif *msg.NotificationMessage)

	// Set the handler of notifications emitted by this module.
	// Intended to create a chain of notifications from producer to consumer.
	//
	// Optionally set additional notification handlers for specific ThingIDs.
	// If a handler for a thingID already exists a warning will be logged and the existing
	// handler will be replaced.
	//
	// thingIDs are the things to handle the notifications for, or empty for all things
	//
	// This can be invoked before or after Start()
	SetNotificationSink(consumer IHiveModule, thingIDs ...string)

	// SetRequestSink sets the handler of requests emitted by this module.
	//
	// This can be invoked before or after Start() to allow for live rewiring of the
	// module chain.
	SetRequestSink(sink IHiveModule)

	// Start readies the module for use.
	// Intended for modulues to initialize resources
	Start() error

	// Stop halts module operation and releases resources.
	// Intended for modulues to free resources
	Stop()
}
