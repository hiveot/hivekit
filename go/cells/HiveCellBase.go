package cells

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/teris-io/shortid"
)

// Cell environment
type CellEnv struct {
	// Application home directory
	HomeDirectory string
	// Application storage directory
	StorageDirectory string
}

// HiveCellBase implements the boilerplate of operating a cell.
// This implements the IHiveCell interface.
// - define and store properties
// - manage message sinks
// - generate TD
// - send notifications for property changes and events
type HiveCellBase struct {

	// cellID is the instance ID of this cell. Used as the senderID in notifications
	// and in logging.
	cellID string

	// notificationSink is the sink for forwarding notification messages to registered cells.
	// sinks set with an empty thingID receive all notifications.
	// sinks set with a specific thingID will receive notifications for that thingID only.
	notificationSinks map[string]api.IHiveCell

	// mutex to access properties
	mux sync.RWMutex

	// requestSink is the cell to forward requests messages to
	requestSink api.IHiveCell

	rpcTimeout time.Duration
}

// ForwardNotification (output) passes received notifications to a registered sink.
//
// If no sinks are registered this does nothing.
func (m *HiveCellBase) ForwardNotification(notif *msg.NotificationMessage) {
	m.mux.RLock()
	sink1, _ := m.notificationSinks[notif.ThingID]
	sink2, _ := m.notificationSinks[""]
	m.mux.RUnlock()

	// first notify the thingID registered cell
	if notif.ThingID != "" && sink1 != nil {
		sink1.HandleNotification(notif)
	}
	// next the generic sink
	if sink2 != nil {
		sink2.HandleNotification(notif)
	}
}

// ForwardRequest passes the request to the sink's HandleRequest method.
// If no sink os configured this returns an not-deliverable error
func (m *HiveCellBase) ForwardRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	m.mux.RLock()
	sink := m.requestSink
	m.mux.RUnlock()
	if sink == nil {
		// end of the line
		return fmt.Errorf("ForwardRequest: request '%s/%s' to thingID '%s' is undeliverable by cell '%s'",
			req.Operation, req.Name, req.ThingID, m.cellID)
	}
	if replyTo == nil {
		slog.Warn("ForwardRequest: no replyTo handler provided",
			"cellID", m.cellID, "req.Sender", req.SenderID, "req.ThingID", req.ThingID)
	}
	err = sink.HandleRequest(req, replyTo)
	return err
}

// ForwardRequestWait is a helper function to pass a request to the sink and wait for a response.
// If no sink os configured this returns an error.
// If the response contains an error, that error is also returned.
func (m *HiveCellBase) ForwardRequestWait(req *msg.RequestMessage) (
	resp *msg.ResponseMessage, err error) {

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
	} else {
		slog.Error("ForwardRequestWait failed", "me", m.GetThingID(),
			"op", req.Operation,
			"thingID", req.ThingID,
			"name", req.Name, "err", err.Error())
	}
	return resp, err
}

// GetNotificationSink returns the default notification sink
// (the one without thingID)
func (m *HiveCellBase) GetNotificationSink() api.IHiveCell {
	m.mux.RLock()
	defer m.mux.RUnlock()
	sink, _ := m.notificationSinks[""]
	return sink
}

// GetRequestSink returns the cell's request sink
func (m *HiveCellBase) GetRequestSink() api.IHiveCell {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return m.requestSink
}

// GetThingID returns the cell's thingID
func (m *HiveCellBase) GetThingID() string {
	return m.cellID
}

// GetTimeout returns the cell's rpc timeout
func (m *HiveCellBase) GetTimeout() time.Duration {
	return m.rpcTimeout
}

// HandleNotification receives an incoming notification from a producer.
//
// The default behavior is to passes the notification upstream to the notification sink, if set.
func (m *HiveCellBase) HandleNotification(notif *msg.NotificationMessage) {
	// the reason for the extra indirection is to ensure we're receiving the notification
	// independently from when someone sets a custome notification handler.
	// ForwardNotification will invoke the hook.
	m.ForwardNotification(notif)
}

// HandleRequest handles request for this cell.
//
// This is just the default implementation that forwards the request downstream.
func (m *HiveCellBase) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
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
func (m *HiveCellBase) Rpc(
	operation, thingID, name string, input any, output any) error {

	var resp *msg.ResponseMessage
	req := msg.NewRequestMessage(operation, thingID, name, input)

	resp, err := m.ForwardRequestWait(req)

	if err == nil && resp != nil {
		err = resp.Decode(output)
		if err != nil {
			err = fmt.Errorf("Rpc: Received response for op/thing/name '%s/%s/%s' but can't decode it: %w",
				operation, thingID, name, err)
		}
	}
	return err
}

// // Set the hook to invoke with received notifications
// func (m *HiveCellBase) SetAppNotificationHook(hook msg.NotificationHandler) {
// 	m.mux.Lock()
// 	defer m.mux.Unlock()
// 	m.appNotificationHook = hook
// }

// Set the handler that will receive notifications emitted by this cell.
// Use thingIDs to set an additional handler specific for the specified thingIDs
func (m *HiveCellBase) SetNotificationSink(sink api.IHiveCell, thingIDs ...string) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if len(thingIDs) == 0 {
		thingIDs = []string{""}
	}
	// report missing initialization instead of a nil error
	if m.notificationSinks == nil {
		panic("HiveCellBase.SetNotificationSink. This cell is not initialized")
	}

	for _, thingID := range thingIDs {
		if m.notificationSinks[thingID] != nil {
			slog.Warn("SetNotificationSink: A notification sink already exists. It will be overwritten.",
				"cellID", m.cellID,
				"thingID", thingID)
		}
		m.notificationSinks[thingID] = sink
	}
}

// SetRequestSink sets the handler for requests send or forwarded by this cell.
//
//	requestSink is the sink that will handle requests and send notifications
func (m *HiveCellBase) SetRequestSink(requestSink api.IHiveCell) {
	m.mux.Lock()
	defer m.mux.Unlock()
	// to be determined if there is a use-case for replacing the sink
	if m.requestSink != nil {
		slog.Warn("SetRequestSink: Overriding existing request sink",
			"cellID", m.GetThingID())
	}
	m.requestSink = requestSink
}

// // SetTimeout changes the timeout when waiting for result.
func (m *HiveCellBase) SetTimeout(rpcTimeout time.Duration) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.rpcTimeout = rpcTimeout
}

// Start the cell .. owning struct must implement this
func (co *HiveCellBase) Start() error { return nil }

// Stop the cell .. owning struct must implement this
func (co *HiveCellBase) Stop() {}

// Create a new cell base
//
//	cellID is the instance ID of the cell. "" to auto generate.
//	timeout for forwarding request and waiting for the result
func NewHiveCellBase(cellID string, rpcTimeout time.Duration) *HiveCellBase {
	if rpcTimeout == 0 {
		rpcTimeout = msg.DefaultRnRTimeout
	}
	if cellID == "" {
		cellID = "thing-" + shortid.MustGenerate()
	}
	m := &HiveCellBase{
		mux:               sync.RWMutex{},
		cellID:            cellID,
		rpcTimeout:        rpcTimeout,
		notificationSinks: make(map[string]api.IHiveCell),
	}
	return m
}
