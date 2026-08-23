package api

import (
	"github.com/hiveot/hivekit/go/api/msg"
)

// The HiveOT cell interface
// Anything that accepts requests can be a cell, including clients and servers.
// This interface is the most basic cell interface.
type IHiveCell interface {

	// GetNotificationSink returns the cell's notification sink that was set with SetRequestSink
	GetNotificationSink() IHiveCell

	// GetRequestSink returns the cell's request sink that was set with SetRequestSink
	GetRequestSink() IHiveCell

	// GetThingID returns the cell's instance ID.
	// This is used as the sender ThingID when sending notifications.
	GetThingID() string

	// HandleRequest processes or forwards the request.
	//
	// When the request is for this cell then the cell processes the request and
	// invokes replyTo with the response. ReplyTo is invoked asynchronously before
	// or after returning.
	//
	// When the request is not for this cell then it is forwarded:
	//
	// 1. By default cell forward unhandled requests to their request sink.
	//    Flow: consumer -> cell -[rsink]-> producer
	//
	// 2. If the cell is a transport client: the request is transported to the server,
	//    and the server passes it to the producer that is registered as its sink.
	//    Flow: consumer -[rsink]-> tp-client -> tp-server -[rsink]-> producer
	//
	// 3. If the cell is a transport server or server connection then the request is
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
	// A middleware cell can intercept the response by forwarding the request downstream
	// while providing its own handler as the replyTo. This handler then forwards the response
	// to the original replyTo endpoint.
	//
	// This returns an error if the provided replyTo will not be able to receive a response.
	//
	//  request is the request to process or forward
	//  replyTo is the response callback. This MUST be called if the request has been processed.
	HandleRequest(request *msg.RequestMessage, replyTo msg.ResponseHandler) error

	// Handle the notification received from a producer.
	// The default behavior is to forward it upstream to the handler set with SetNotificationSink.
	HandleNotification(notif *msg.NotificationMessage)

	// Set the handler of notifications emitted by this cell.
	// Intended to create a chain of notifications from producer to consumer.
	//
	// Optionally set additional notification handlers for specific ThingIDs.
	// If a handler for a thingID already exists a warning will be logged and the existing
	// handler will be replaced.
	//
	// thingIDs are the things to handle the notifications for, or empty for all things
	//
	// This can be invoked before or after Start()
	SetNotificationSink(consumer IHiveCell, thingIDs ...string)

	// SetRequestSink sets the handler of requests emitted by this cell.
	//
	// This can be invoked before or after Start() to allow for live rewiring of the
	// cell chain.
	SetRequestSink(sink IHiveCell)

	// Start readies the cell for use.
	//
	// Note that during Start cells might not yet be linked, so they must not send
	// any requests until all cells have started.
	//
	// If this is an issue then try to solve it by providing dependencies during
	// instantiation, not during Start. When using the factory, other cells can be
	// retrieved using f.GetCell(type).(interface), where f is the factory instance.
	//
	// If the cell cannot be used as intended then return an error.
	Start() error

	// Stop halts cell operation and releases resources.
	Stop()
}
