package internal

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketstorepkg "github.com/hiveot/hivekit/go/modules/bucketstore/pkg"
	"github.com/hiveot/hivekit/go/modules/history"
)

// HistoryService provides storage for request and notification history.
//
// Requests received are forwarded to the registered sink and stored if they pass the
// filter. Storage is done using the NotificationMessage envelope.
// Similarly, notifications are forwarded as-is and stored if they pass the
// notification filter.
//
// Each Thing has a bucket with events and actions.
// This implements the IHistoryService and IHiveModule interface
type HistoryService struct {
	modules.HiveModuleBase

	// The thingID of the history service for messaging
	historyThingID string

	// The underlying bucketstore instance
	bucketStore bucketstore.IBucketStorage

	config history.HistoryConfig

	// cache of cursors with lifecycle management intended for remote users
	// re-use the one from the bucket store
	cursorCache bucketstore.ICursorCache

	// lifespan of cursor iterator
	cursorLifespan time.Duration

	// RRN message handler for reading history
	readHistoryMsgHandler *ReadHistoryMsgHandler
}

// Forward notifications to the registered sink and record it if they pass the filter.
func (m *HistoryService) HandleNotification(notif *msg.NotificationMessage) {
	go func() {
		if m.config.NotificationFilter.AcceptNotification(notif) {
			m.StoreNotification(notif)
		}
	}()
	m.ForwardNotification(notif)
}

// HandleRequest handles request for this module.
// If not a module request then record it in the history store if it passes
// the filters and forward the request to the registered sink.
func (m *HistoryService) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	if req.ThingID == m.historyThingID {
		// handle requests for the history service itself
		err := m.readHistoryMsgHandler.HandleRequest(req, replyTo)
		return err
	}
	go func() {
		if m.config.RequestFilter.AcceptRequest(req) {
			m.StoreRequest(req)
		}
	}()
	return m.ForwardRequest(req, replyTo)
}

// Start the history module and open the store
// this loads the filters
func (m *HistoryService) Start() (err error) {
	m.bucketStore, err = bucketstorepkg.OpenBucketStore(m.config.StoreDirectory, m.config.Backend)
	if err != nil {
		return err
	}

	slog.Info("Start: Starting history module with backend " + m.config.Backend)
	// Messaging API handler for reading the history
	m.readHistoryMsgHandler = NewReadHistoryMsgHandler(m)

	return err
}

// Stop using the history service and release resources
func (m *HistoryService) Stop() {
	slog.Info("Stop: Stopping history module")
	_ = m.bucketStore.Close()
}

// Store notifications for later retrieval
func (m *HistoryService) StoreNotification(notif *msg.NotificationMessage) error {
	err := m.AddValue(notif)
	return err
}

// Store requests for later retrieval
func (m *HistoryService) StoreRequest(req *msg.RequestMessage) error {

	if req.Operation != td.OpInvokeAction {
		return fmt.Errorf("AddAction: Operation is not invokeaction")
	}
	// convert the notification to a ThingValue for storage
	value := msg.NewNotificationMessage(
		req.SenderID,
		msg.AffordanceTypeAction,
		req.ThingID,
		req.Name,
		req.Input,
	)
	value.Timestamp = req.Timestamp
	err := m.AddValue(value)
	return err
}

// NewHistoryService creates a new instance for the history module using the given
// configuration.
//
// A configuration can be created using: config.NewHistoryConfig(storeDirectory, backend)
func NewHistoryService(config history.HistoryConfig) *HistoryService {

	m := &HistoryService{
		historyThingID: history.DefaultHistoryThingID,
		cursorLifespan: time.Minute,
		cursorCache:    bucketstorepkg.NewCursorCache(),
		config:         config,
	}
	// m.config = NewHistoryConfig()
	// m.config = config.NewHistoryConfig(storeDirectory, backend)

	var _ history.IHistoryService = m // interface check
	return m
}
