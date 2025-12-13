package modules

import "github.com/hiveot/hivekit/go/modules/messaging"

// The golang pipeline module interface  (proposed)
//
// The module is responsesible for creating the optional TM, receive requests and
// sending notifications.
type IHiveModule interface {
	// GetGetTMTD returns the module's Thing Model describing its properties, actions and events.
	// If no TD is supported then this returns nil.
	// Forms in the TD are typically added by the pipeline messaging server.
	GetTM() string

	// HandleRequest processes or forwards an SME request message.
	// This returns an SME response containing the delivery status and result.
	HandleRequest(request *messaging.RequestMessage) *messaging.ResponseMessage

	// HandleNotification processes or forwards an SME notification message.
	// Notification messages consists of subscribed events and observed properties.
	HandleNotification(notif *messaging.NotificationMessage)

	// AddSink sets the destination sink to forward messages to, to send the processing result to, or both.
	// Modules can support a single or multiple sinks. If no more sinks can be added an error is returned.
	// AddSink can be invoked before or after start is called.
	AddSink(sink IHiveModule) error

	// Start readies the module for use
	Start() error

	// Stop halts module operation and releases resources.
	Stop()
}
