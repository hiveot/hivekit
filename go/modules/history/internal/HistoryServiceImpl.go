package internal

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/modules/bucketstore/kvbtreestore"
	"github.com/hiveot/hivekit/go/modules/bucketstore/pebblestore"
	bucketstoreservice "github.com/hiveot/hivekit/go/modules/bucketstore/service"
	"github.com/hiveot/hivekit/go/modules/history"
)

// HistoryServiceImpl provides storage for request and notification history.
//
// Requests received are forwarded to the registered sink and stored if they pass the
// filter. Storage is done using the NotificationMessage envelope.
// Similarly, notifications are forwarded as-is and stored if they pass the
// notification filter.
//
// Each Thing has a bucket with events and actions.
// This implements the IHistoryService and IHiveModule interface
type HistoryServiceImpl struct {
	*modules.HiveModuleBase

	// The underlying bucketstore instance
	bucketStore bucketstore.IBucketStore

	config history.HistoryConfig

	// cache of cursors with lifecycle management intended for remote users
	// re-use the one from the bucket store
	cursorCache bucketstore.ICursorCache

	// lifespan of cursor iterator
	cursorLifespan time.Duration
}

// Forward notifications to the registered sink and record it if they pass the filter.
func (m *HistoryServiceImpl) HandleNotification(notif *msg.NotificationMessage) {
	go func() {
		if m.config.NotificationFilter.AcceptNotification(notif) {
			m.StoreNotification(notif)
		}
	}()
	m.ForwardNotification(notif)
}

// Start the history module and open the store
// this loads the filters
func (m *HistoryServiceImpl) Start() (err error) {
	switch m.config.Backend {
	case bucketstore.BackendPebble:
		m.bucketStore = pebblestore.NewBucketStore(m.config.StoreDirectory)
		err = m.bucketStore.Open()
	case bucketstore.BackendKVBTree:
		m.bucketStore = kvbtreestore.NewBucketStore(m.config.StoreDirectory)
		err = m.bucketStore.Open()
	default:
		err = fmt.Errorf("Start: Unknown bucket store backend type '%s'", m.config.Backend)
	}
	if err != nil {
		return err
	}

	slog.Info("Start: Starting history module with backend " + m.config.Backend)
	// Messaging API handler for reading the history
	// m.readHistoryMsgHandler = NewReadHistoryMsgHandler(m)

	return err
}

// Stop using the history service and release resources
func (m *HistoryServiceImpl) Stop() {
	slog.Info("Stop: Stopping history module")
	_ = m.bucketStore.Close()
}

// Store notifications for later retrieval
func (m *HistoryServiceImpl) StoreNotification(notif *msg.NotificationMessage) error {
	err := m.AddValue(notif)
	return err
}

// Store requests for later retrieval
func (m *HistoryServiceImpl) StoreRequest(req *msg.RequestMessage) error {

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

// NewHistoryServiceImpl creates a new instance for the history module using the given
// configuration.
//
// A configuration can be created using: config.NewHistoryConfig(storeDirectory, backend)
func NewHistoryServiceImpl(config history.HistoryConfig) *HistoryServiceImpl {

	thingID := history.DefaultHistoryThingID
	m := &HistoryServiceImpl{
		HiveModuleBase: modules.NewHiveModuleBase(thingID, 0),
		cursorLifespan: time.Minute,
		cursorCache:    bucketstoreservice.NewCursorCache(),
		config:         config,
	}
	// m.config = NewHistoryConfig()
	// m.config = config.NewHistoryConfig(storeDirectory, backend)

	var _ history.IHistoryService = m // interface check
	return m
}
