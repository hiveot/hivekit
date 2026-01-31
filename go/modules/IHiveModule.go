package modules

import (
	"github.com/hiveot/hivekit/go/msg"
)

// The HiveOT module interface
// Anything that accepts requests can be a module, including clients and servers.
// This interface is the most basic module interface.
type IHiveModule interface {
	// GetModuleID returns module's ID.
	// For agents/devices this is the ThingID, for consumers this is the clientID.
	GetModuleID() string

	// GetTM returns the module's [W3C WoT Thing Model](https://www.w3.org/TR/wot-thing-description11/#thing-model)
	// in JSON, describing its properties, actions and events.
	//
	// The TM is converted to a TD using transport modules 'AddForms' which adds forms that describe
	// the interaction using the given transport. The TD is then published in a directory
	// service that is available to consumers. This is typically the responsibility of the pipeline
	// service. If the pipeline service is not used then this is up to the application startup logic.
	//
	// Only actual producers of information need to implement a TM. The TM can be obtained
	// after a successful start. If the module does not support a TM then this
	// returns an empty string.
	//
	// The HiveModuleBase implements a default method returning an empty TM.
	GetTM() string

	// HandleRequest - invoked by consumer to this producer.
	//  [producer] processes or forwards a request downstream to other producers.
	//
	// When the request is for this module then the module processes the request and
	// invokes replyTo with the response. ReplyTo is invoked asynchronously before
	// or after returning.
	//
	// When the request is not for this producer then it is forwarded:
	//
	// 1. Most modules forward the request to their sink which is the linked downstream producer.
	//    Flow: consumer -[sink]-> producer
	//
	// 2. If the module is a transport client: the request is transported to the server,
	//    and the server passes it to the producer that is registered as its sink.
	//    Flow: consumer -[sink]-> client -> server -[sink]-> producer
	//
	// 3. If the module is a transport server or server connection then the request is
	//    transported to the remote client. The client passes it to its registered sink.
	//    This sink should be a producer that can handle the request.
	//    (In this case the consumer is a process running on the server)
	//    Flow: consumer -[sink]-> server -> client -[sink]-> producer
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

	// Handle the notification received from the producer.
	// The default behavior is to forward it upstream to the handler set with SetNotificationSink.
	HandleNotification(notif *msg.NotificationMessage)

	// Set the handler of notifications emitted by this module (acting as a producer)
	// Intended to create a chain of notifications from producer to consumer.
	//
	// This can be invoked before or after Start()
	SetNotificationSink(sink msg.NotificationHandler)

	// SetRequestSink sets the producer that will handle the requests emitted by this module.
	//
	// This can be invoked before or after Start() to allow for live rewiring of the
	// module chain.
	SetRequestSink(sink msg.RequestHandler)

	// Start readies the module for use.
	// Intended for modulues to initialize resources
	//  yamlConfig is an optional configuration or "" if not used
	Start(yamlConfig string) error

	// Stop halts module operation and releases resources.
	// Intended for modulues to free resources
	Stop()
}
