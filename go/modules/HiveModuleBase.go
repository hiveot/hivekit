package modules

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/teris-io/shortid"
)

// Module application environment
type ModuleEnv struct {
	// Application home directory
	HomeDirectory string
	// Application storage directory
	StorageDirectory string
}

// HiveModuleBase implements the boilerplate of running a module.
// This implements the IHiveModule interface.
// - define and store properties
// - manage message sinks
// - generate TD
// - send notifications for property changes and events
//
// Call Init(moduleID,sink) after construction
type HiveModuleBase struct {

	// thingID is the instance ID of this module. Used as the senderID in notifications
	// and in logging.
	thingID string

	// notificationSink is the sink for forwarding notification messages
	// This is the upstream consumer.
	notificationSink msg.NotificationHandler

	// module properties and their value, nil if not used
	// use UpdateProperty to modify a value and flag it for change
	// properties map[string]any

	// mutex to access properties
	mux sync.RWMutex

	// requestSink is the sink for forwarding requests messages to
	requestSink msg.RequestHandler

	rpcTimeout time.Duration

	// the senderID for requests. Intended to hold the authenticated clientID.
	// Client side modules can use their moduleID for use in logging.
	// senderID string
}

// ForwardNotification (output) passes received notifications to a registered hook
// and send it to the a registered sink.
//
// Note that only handleNotification passes it to the appNotificationHook.
//
// If none is registered this does nothing.
// note that the handler is not the downstream sink but the upstream consumer.
func (m *HiveModuleBase) ForwardNotification(notif *msg.NotificationMessage) {
	m.mux.RLock()
	handler := m.notificationSink
	m.mux.RUnlock()
	if handler == nil {
		// End of the line. If the notification isn't handled then warn about it
		// A downstream module could have subscribed.
		// // keep this warning for now.
		// slog.Info("ForwardNotification: end of the line, no more notification sink.",
		// 	"module", fmt.Sprintf("%T", m),
		// 	"affordance", notif.AffordanceType,
		// 	"thingID", notif.ThingID,
		// 	"name", notif.Name,
		// )
		return
	}
	handler(notif)
}

// ForwardRequest passes the request to the sink's HandleRequest method.
// If no sink os configured this returns an error
// This assigns a request correlationID if none is set.
func (m *HiveModuleBase) ForwardRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	m.mux.RLock()
	handler := m.requestSink
	m.mux.RUnlock()
	if handler == nil {
		return fmt.Errorf("ForwardRequest: end of the line at '%s' for request '%s/%s' to thingID '%s'",
			m.thingID, req.Operation, req.Name, req.ThingID)
	}
	if replyTo == nil {
		slog.Info("ForwardRequest: no replyTo handler provided", "moduleID", m.thingID)
	}
	err = handler(req, replyTo)
	return err
}

// ForwardRequestWait is a helper function to pass a request to the sink and wait for a response.
// If no sink os configured this returns an error.
// If the response contains an error, that error is also returned.
func (m *HiveModuleBase) ForwardRequestWait(
	req *msg.RequestMessage) (resp *msg.ResponseMessage, err error) {

	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}

	ar := utils.NewAsyncReceiver[*msg.ResponseMessage]()
	err = m.ForwardRequest(req, func(r *msg.ResponseMessage) error {
		ar.SetResponse(r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	timeout := m.rpcTimeout
	if timeout == 0 {
		timeout = msg.DefaultRnRTimeout
	}
	resp, err = ar.WaitForResponse(timeout)
	if err == nil {
		err = resp.AsError()
	}
	return resp, err
}

// GetThingID returns the module's thingID
func (m *HiveModuleBase) GetThingID() string {
	return m.thingID
}

// GetTimeout returns the module's rpc timeout
func (m *HiveModuleBase) GetTimeout() time.Duration {
	return m.rpcTimeout
}

// // GetSink returns the module's request sink
// func (m *HiveModuleBase) GetSink() msg.RequestHandler {
// 	m.mux.RLock()
// 	defer m.mux.RUnlock()
// 	return m.requestSink
// }

// HandleNotification receives an incoming notification from a producer.
//
// The default behavior is to passes the notification upstream to the notification sink, if set.
func (m *HiveModuleBase) HandleNotification(notif *msg.NotificationMessage) {
	// the reason for the extra indirection is to ensure we're receiving the notification
	// independently from when someone sets a custome notification handler.
	// ForwardNotification will invoke the hook.
	m.ForwardNotification(notif)
}

// HandleRequest handles request for this module.
//
// This is just the default implementation that forwards the request downstream.
func (m *HiveModuleBase) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	return m.ForwardRequest(req, replyTo)
}

// Rpc is a convenience function to create and send a request message and decode the a response.
// This returns an error if the request fails or if the response contains an error
//
//	operation is the WoT operation to send
//	thingID is the Thing to address
//	name is the operation name as defined in the TD
//	input are optional input parameters or nil if none
//	output is a pointer to the  struct where the result will be decoded
func (m *HiveModuleBase) Rpc(
	operation, thingID, name string, input any, output any) error {

	var resp *msg.ResponseMessage
	req := msg.NewRequestMessage(operation, thingID, name, input)

	resp, err := m.ForwardRequestWait(req)

	if err == nil && resp != nil {
		err = resp.Decode(output)
	}
	return err
}

// // Set the hook to invoke with received notifications
// func (m *HiveModuleBase) SetAppNotificationHook(hook msg.NotificationHandler) {
// 	m.mux.Lock()
// 	defer m.mux.Unlock()
// 	m.appNotificationHook = hook
// }

// Set the handler that will receive notifications emitted by this module
func (m *HiveModuleBase) SetNotificationSink(consumer msg.NotificationHandler) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.notificationSink != nil {
		slog.Warn("SetNotificationSink: A notification sink already exists. It will be overwritten.",
			"moduleID", m.thingID)
	}
	m.notificationSink = consumer
}

// SetRequestSink sets the producer that will handle requests for this consumer and register this
// module as the receive of notifications from the module.
//
//	producer is the sink that will handle requests and send notifications
func (m *HiveModuleBase) SetRequestSink(sink msg.RequestHandler) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.requestSink = sink
}

// // SetTimeout changes the timeout when waiting for result.
func (m *HiveModuleBase) SetTimeout(rpcTimeout time.Duration) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.rpcTimeout = rpcTimeout
}

// Start the consumer module .. owning struct must implement this
func (co *HiveModuleBase) Start() error { return nil }

// Stop the consumer module .. owning struct must implement this
func (co *HiveModuleBase) Stop() {}

// Create a new module
//
// the thingID is required when this module implements HandleRequest.
//
//	thingID is the instance ID of the module. "" to auto generate.
//	timeout for forwarding request and waiting for the result
func NewHiveModuleBase(thingID string, rpcTimeout time.Duration) *HiveModuleBase {
	if rpcTimeout == 0 {
		rpcTimeout = msg.DefaultRnRTimeout
	}
	if thingID == "" {
		thingID = "thing-" + shortid.MustGenerate()
	}
	m := &HiveModuleBase{
		mux:        sync.RWMutex{},
		thingID:    thingID,
		rpcTimeout: rpcTimeout,
	}
	return m
}
