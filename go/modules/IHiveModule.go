package modules

import (
	"github.com/hiveot/hivekit/go/lib/messaging"
	"github.com/hiveot/hivekit/go/wot/td"
)

// The golang pipeline module interface  (proposed)
//
// The module is responsesible for creating the optional TM, receive requests and
// sending notifications.
type IHiveModule interface {
	// GetTD returns the module's TD describing its properties, actions and events.
	// If supported, the TD can be obtained after a successful start.
	// If no TD is supported then this returns nil.
	// Forms in the TD are typically added by the pipeline messaging server.
	GetTD() *td.TD

	// HandleRequest processes or forwards a request message. This returns a response
	// containing the delivery status and optionally a result.
	HandleRequest(request *messaging.RequestMessage) *messaging.ResponseMessage

	// HandleNotification processes or forwards a notification message.
	// Notification messages consists of subscribed events and observed properties.
	HandleNotification(*messaging.NotificationMessage)

	// AddSink sets the destination sink to forward messages to, to send the processing result to, or both.
	// Modules can support a single or multiple sinks. If no more sinks can be added an error is returned.
	// AddSink can be invoked before or after start is called.
	AddSink(sink IHiveModule) error

	// Start readies the module for use using the given yaml configuration.
	// Start must be invoked before passing messages.
	Start(yamlConfig string) error
	Stop()
}
