package module

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketserver "github.com/hiveot/hivekit/go/modules/bucketstore/server"
	"github.com/hiveot/hivekit/go/modules/bucketstore/stores"
	"github.com/hiveot/hivekit/go/modules/history"
	"github.com/hiveot/hivekit/go/modules/history/config"
	historyserver "github.com/hiveot/hivekit/go/modules/history/server"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
	"go.yaml.in/yaml/v2"
)

// HistoryService provides storage for request and notification history.
//
// Requests received are forwarded to the registered sink and stored if they pass the filter.
// Similarly, notifications are forwarded as-is and stored if they pass the notification filter.
//
// Each Thing has a bucket with events and actions.
// This implements the IHistoryService interface
type HistoryModule struct {
	modules.HiveModuleBase

	// The underlying bucketstore instance
	bucketStore bucketstore.IBucketStore

	config config.HistoryConfig

	// cache of cursors with lifecycle management intended for remote users
	// re-use the one from the bucket store
	cursorCache *bucketserver.CursorCache

	// lifespan of cursor iterator
	cursorLifespan time.Duration

	// RRN message handler for reading history
	readHistoryMsgHandler *historyserver.ReadHistoryMsgHandler
}

// Forward notifications to the registered sink and record it if they pass the filter.
func (m *HistoryModule) HandleNotification(notif *msg.NotificationMessage) {
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
func (m *HistoryModule) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	if req.ThingID == historyserver.ReadHistoryServiceID {
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
func (m *HistoryModule) Start(yamlConfig string) (err error) {
	if yamlConfig != "" {
		err = yaml.Unmarshal([]byte(yamlConfig), &m.config)
		if err != nil {
			slog.Error("Start: Failed to load history service config", "error", err)
			return err
		}
	}
	m.SetModuleID(m.config.ModuleID)

	m.bucketStore, err = stores.OpenBucketStore(m.config.StoreDirectory, m.config.Backend)
	if err != nil {
		return err
	}

	slog.Info("Starting HistoryService", "moduleID", m.GetModuleID())
	// Messaging API handler for reading the history
	m.readHistoryMsgHandler = historyserver.NewReadHistoryMsgHandler(m)

	return err
}

// Stop using the history service and release resources
func (m *HistoryModule) Stop() {
	slog.Info("Stopping HistoryService")
	_ = m.bucketStore.Close()
}

// Store notifications for later retrieval
func (m *HistoryModule) StoreNotification(notif *msg.NotificationMessage) error {

	// convert the notification to a ThingValue for storage
	tv := msg.NewThingValue(
		notif.SenderID,
		notif.AffordanceType,
		notif.ThingID,
		notif.Name,
		notif.Data,
		notif.Timestamp,
	)
	err := m.AddValue(tv)
	return err
}

// Store notifications for later retrieval
func (m *HistoryModule) StoreRequest(req *msg.RequestMessage) error {

	if req.Operation != wot.OpInvokeAction {
		return fmt.Errorf("AddAction: Operation is not invokeaction")
	}
	// convert the notification to a ThingValue for storage
	tv := msg.NewThingValue(
		req.SenderID,
		msg.AffordanceTypeAction,
		req.ThingID,
		req.Name,
		req.Input,
		req.Created,
	)
	err := m.AddValue(tv)
	return err
}

// NewHistoryModule creates a new instance for the history module using the given
// storage bucket.
func NewHistoryModule(storeDirectory string, backend string) *HistoryModule {

	m := &HistoryModule{
		cursorLifespan: time.Minute,
		cursorCache:    bucketserver.NewCursorCache(),
	}
	// m.config = NewHistoryConfig()
	m.config = config.NewHistoryConfig(storeDirectory, backend)

	var _ history.IHistoryModule = m // interface check
	return m
}
