package historyserver

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketstoreapi "github.com/hiveot/hivekit/go/modules/bucketstore/api"
	historyapi "github.com/hiveot/hivekit/go/modules/history/api"
	"github.com/hiveot/hivekit/go/modules/history/config"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
	"go.yaml.in/yaml/v2"
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
type HistoryServer struct {
	modules.HiveModuleBase

	// The underlying bucketstore instance
	bucketStore bucketstoreapi.IBucketStore

	config config.HistoryConfig

	// cache of cursors with lifecycle management intended for remote users
	// re-use the one from the bucket store
	cursorCache *bucketstore.CursorCache

	// lifespan of cursor iterator
	cursorLifespan time.Duration

	// RRN message handler for reading history
	readHistoryMsgHandler *ReadHistoryMsgHandler
}

// Forward notifications to the registered sink and record it if they pass the filter.
func (m *HistoryServer) HandleNotification(notif *msg.NotificationMessage) {
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
func (m *HistoryServer) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	if req.ThingID == m.GetModuleID() {
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
func (m *HistoryServer) Start(yamlConfig string) (err error) {
	if yamlConfig != "" {
		err = yaml.Unmarshal([]byte(yamlConfig), &m.config)
		if err != nil {
			slog.Error("Start: Failed to load history service config", "error", err)
			return err
		}
	}
	m.SetModuleID(m.config.ModuleID)

	m.bucketStore, err = bucketstore.OpenBucketStore(m.config.StoreDirectory, m.config.Backend)
	if err != nil {
		return err
	}

	slog.Info("Starting HistoryService", "moduleID", m.GetModuleID())
	// Messaging API handler for reading the history
	m.readHistoryMsgHandler = NewReadHistoryMsgHandler(m)

	return err
}

// Stop using the history service and release resources
func (m *HistoryServer) Stop() {
	slog.Info("Stopping HistoryService")
	_ = m.bucketStore.Close()
}

// Store notifications for later retrieval
func (m *HistoryServer) StoreNotification(notif *msg.NotificationMessage) error {
	err := m.AddValue(notif)
	return err
}

// Store requests for later retrieval
func (m *HistoryServer) StoreRequest(req *msg.RequestMessage) error {

	if req.Operation != wot.OpInvokeAction {
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
	value.Timestamp = req.Created
	err := m.AddValue(value)
	return err
}

// NewHistoryServer creates a new instance for the history module using the given
// storage bucket.
func NewHistoryServer(storeDirectory string, backend string) *HistoryServer {

	m := &HistoryServer{
		cursorLifespan: time.Minute,
		cursorCache:    bucketstore.NewCursorCache(),
	}
	// m.config = NewHistoryConfig()
	m.config = config.NewHistoryConfig(storeDirectory, backend)

	var _ historyapi.IHistoryServer = m // interface check
	return m
}
