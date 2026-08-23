package internal

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/bucketstore"
	"github.com/hiveot/hivekit/go/cells/bucketstore/kvbtreestore"
	"github.com/hiveot/hivekit/go/cells/bucketstore/pebblestore"
	bucketstoreservice "github.com/hiveot/hivekit/go/cells/bucketstore/service"
	"github.com/hiveot/hivekit/go/cells/history"
)

// HistoryServiceImpl provides storage for request and notification history.
//
// Requests received are forwarded to the registered sink and stored if they pass the
// filter. Storage is done using the NotificationMessage envelope.
// Similarly, notifications are forwarded as-is and stored if they pass the
// notification filter.
//
// Each Thing has a bucket with events and actions.
// This implements the IHistoryService and IHiveCell interface
type HistoryServiceImpl struct {
	*cells.HiveCellBase

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
func (svc *HistoryServiceImpl) HandleNotification(notif *msg.NotificationMessage) {
	go func() {
		if svc.config.NotificationFilter.AcceptNotification(notif) {
			svc.StoreNotification(notif)
		}
	}()
	svc.ForwardNotification(notif)
}

// Start the history service and open the store
// this loads the filters
func (svc *HistoryServiceImpl) Start() (err error) {
	switch svc.config.Backend {
	case bucketstore.BackendPebble:
		svc.bucketStore = pebblestore.NewBucketStore(svc.config.StoreDirectory)
		err = svc.bucketStore.Open()
	case bucketstore.BackendKVBTree:
		svc.bucketStore = kvbtreestore.NewBucketStore(svc.config.StoreDirectory)
		err = svc.bucketStore.Open()
	default:
		err = fmt.Errorf("Start: Unknown bucket store backend type '%s'", svc.config.Backend)
	}
	if err != nil {
		return err
	}

	slog.Info("Start: Starting history service with backend " + svc.config.Backend)
	// Messaging API handler for reading the history
	// m.readHistoryMsgHandler = NewReadHistoryMsgHandler(m)

	return err
}

// Stop using the history service and release resources
func (svc *HistoryServiceImpl) Stop() {
	slog.Info("Stop: Stopping history service")
	_ = svc.bucketStore.Close()
}

// Store notifications for later retrieval
func (svc *HistoryServiceImpl) StoreNotification(notif *msg.NotificationMessage) error {
	err := svc.AddValue(notif)
	return err
}

// Store requests for later retrieval
func (svc *HistoryServiceImpl) StoreRequest(req *msg.RequestMessage) error {

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
	err := svc.AddValue(value)
	return err
}

// NewHistoryServiceImpl creates a new instance for the history service using the given
// configuration.
//
// A configuration can be created using: config.NewHistoryConfig(storeDirectory, backend)
func NewHistoryServiceImpl(config history.HistoryConfig) *HistoryServiceImpl {

	thingID := history.DefaultHistoryThingID
	svc := &HistoryServiceImpl{
		HiveCellBase:   cells.NewHiveCellBase(thingID, 0),
		cursorLifespan: time.Minute,
		cursorCache:    bucketstoreservice.NewCursorCache(),
		config:         config,
	}
	// m.config = NewHistoryConfig()
	// m.config = config.NewHistoryConfig(storeDirectory, backend)

	var _ history.IHistoryService = svc // interface check
	return svc
}
