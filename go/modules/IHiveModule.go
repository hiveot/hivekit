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
	// 2b. If a producer running on the server makes a request to another producer then
	//     it acts as a consumer.
	//     The producer has to have the server set as its sink so it can pass requests
	//     to the client serving the producer.
	//
	// 3. The module is a transport server connection: the request is transported to the
	//    connected client, and the client passes it to the producer that is its registered sink.
	//    Flow: consumer -[sink]-> server -> client -[sink]-> producer
	//
	// 3b. If a producer that is connected through a client makes a request to another
	//     producer then it acts as a consumer.
	//     The producer has to have the client set as its sink so it can pass requests
	//     to the server.
	//
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

	// Set the handler of notifications produced (or forwarded) by this module
	// When used in a chain this is the consumer for which this module is the producer.
	//
	// This is typically not set directly by the consumer. Instead a module's SetSink
	// handler calls SetNotification on the sink so that notifications can be received
	// by the consumer calling SetSink.
	SetNotificationHandler(consumer msg.NotificationHandler)

	// SetSink [consumer] sets the given module as the producer for requests and notifications.
	// and assign a handler that receives the notifications from this module.
	//
	// If this module is a transport client then requests received from the remote server
	// are passed to this sink. Notifications received from this producer are passed to the
	// remote server.
	//
	// The notification handler should handle notifications this module is interested in.
	//
	// If no notification handler is provided then the registered notification handler
	// of this module is used, so that notifications are passed up the chain.
	SetSink(producer IHiveModule, notifHandler msg.NotificationHandler)

	// Start readies the module for use.
	// Intended for modulues to initialize resources
	//  yamlConfig is an optional configuration or "" if not used
	Start(yamlConfig string) error

	// Stop halts module operation and releases resources.
	// Intended for modulues to free resources
	Stop()
}
