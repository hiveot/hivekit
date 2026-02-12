package module

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketserver "github.com/hiveot/hivekit/go/modules/bucketstore/server"
	"github.com/hiveot/hivekit/go/modules/history"
	"github.com/hiveot/hivekit/go/modules/history/server"
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

	config history.HistoryConfig

	// cache of cursors with lifecycle management intended for remote users
	// re-use the one from the bucket store
	cursorCache *bucketserver.CursorCache

	// lifespan of cursor iterator
	cursorLifespan time.Duration

	// RRN message handler for reading history
	readHistoryMsgHandler *server.ReadHistoryMsgHandler
}

// Forward notifications to the registered sink and store if they pass the filter.
func (m *HistoryModule) HandleNotification(notif *msg.NotificationMessage) {
	if m.config.NotificationFilter != nil {
		go func() {
			if m.config.NotificationFilter.RetainNotification(notif) {
				m.StoreNotification(notif)
			}
		}()
	}
	m.ForwardNotification(notif)
}

// Forward requests to the registered sink and store if they pass the filter.
func (m *HistoryModule) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	if m.config.NotificationFilter != nil {
		go func() {
			if m.config.RequestFilter.RetainRequest(req) {
				m.StoreRequest(req)
			}
		}()
	}
	return m.ForwardRequest(req, replyTo)
}

// Start the history service
// this loads the filters
func (m *HistoryModule) Start(yamlConfig string) (err error) {

	slog.Info("Starting HistoryService", "moduleID", m.GetModuleID())
	err = yaml.Unmarshal([]byte(yamlConfig), &m.config)
	if err != nil {
		slog.Error("Failed to load history service config", "error", err)
		return err
	}
	// Messaging API handler for reading the history
	m.readHistoryMsgHandler = server.NewReadHistoryMsgHandler(m)
	if err != nil {
		return err
	}

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
// The bucket store is closed when the module is stopped.
func NewHistoryModule(bucketStore bucketstore.IBucketStore) *HistoryModule {

	m := &HistoryModule{
		bucketStore:    bucketStore,
		cursorLifespan: time.Minute,
		cursorCache:    bucketserver.NewCursorCache(),
	}

	var _ history.IHistoryModule = m // interface check
	return m
}
