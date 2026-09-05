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
	// invokes replyTo with the response. ReplyTo can be invoked (a)synchronously,
	// before or after returning.
	//
	// Either replyTo must be called or an error returned. Not both.
	//
	// When the request is not for this cell then it is forwarded if forwarding is enabled:
	//
	// 1. By default cell forward unhandled requests to their request sink.
	//    This can be disabled with 'SetForwarding()'.
	//    Flow: consumer -> cell -[rsink]-> producer
	//
	// 2. If the cell is a transport client: the request is transported to the server,
	//    and the server emits it to its own registered request sink.
	//	  Requests received from the server are emitted to the request sink of the
	//    client cell. The term 'emitted' here means that messages are not affected by
	//    SetForwarding.
	//    Flow: consumer -> client -> server -> request sink
	//    Flow: producer -> server -> client -> request sink
	//
	// 3. If the cell is a transport server then the request is transported
	//    to the remote client. The client emits it to its registered sink.
	//    This sink should be a producer that can handle the request.
	//
	//    Note this is the use-case where a device uses connection reversal to connect
	//         to a server, like a hub or gateway, to serve IoT data. The gateway passes
	//         requests from consumers to the device which is connected as a client.
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
	// Forwarding can be disabled with SetForwarding()
	HandleNotification(notif *msg.NotificationMessage)

	// Ready notifies the cell that the application environment is ready to go.
	//
	// Intended as 'initialization phase 2' where all cells are properly linked
	// and messages will find their destination.
	//
	// Implementing a Ready handler is optional. It is intended for cells to
	// start autonomous operation, such as background tasks, writing a TD,
	// start publishing events, and property updates.
	//
	// Cells must be fully functional and handle requests and notifications even
	// if Ready is not yet called. Messages are only received after the environment
	// is ready and is busy invoking Ready on other cells.
	//
	// By default this does nothing.
	Ready()

	// Set forwarding of notifications or requests to the configured sink.
	//
	// This does not affect notifications or requests emitted by this cell.
	//
	// The default is to forward both notifications and unhandled requests.
	//
	// This does not affect forwarding in transport clients and servers as
	// these always forward messages to the remote side.
	SetForwarding(forwardNotif bool, forwardReq bool)

	// Set the handler of notifications emitted or forwarded by this cell.
	// Intended to create a chain of notifications from producer to consumer.
	//
	// Forwarding can be disabled using SetForwarding.
	//
	// Optionally set additional notification handlers for specific ThingIDs.
	// If a handler for a thingID already exists a warning will be logged and the existing
	// handler will be replaced.
	//
	// thingIDs are the things to handle the notifications for, or empty for all things
	SetNotificationSink(consumer IHiveCell, thingIDs ...string)

	// SetRequestSink sets the handler of requests emitted by this cell.
	SetRequestSink(sink IHiveCell)

	// Stop halts cell operation and releases resources. This should
	// be called to release resources even if Start was never called.
	Stop()
}
