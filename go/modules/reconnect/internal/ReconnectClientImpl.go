package internal

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/reconnect"
)

// ReconnectClientImpl is a module that automatically reconnects a transport client when
// it loses its connection, and restores event and property subscriptions.
//
// If a connection fails repeatedly a backoff time is increased until the set limit.
//
// The transport client must be provided on instantiation.
//
// TBD: instead of providing a transport client can the next module in the request chain
// be used instead?.  This is a use-case for obtaining a downstream module of a type.
type ReconnectClientImpl struct {
	*modules.HiveModuleBase

	// cancel any reconnect attempts.
	// this is nil if not connecting
	cancelFn func()

	// the client connection instance
	conn api.ITransportClient
	//
	maxReconnectAttempts int // 0 for indefinite

	// limit to the reconnect backoff period
	maxBackoffTimeLimit time.Duration

	// mutex to block subscription updates
	mux sync.RWMutex

	// record of subscriptions by key="{op}-{thingID}-{name}"
	subscriptions map[string]*msg.RequestMessage
}

// applySubscription applies recorded subscriptions
// this will lock subscriptions until complete or error
func (m *ReconnectClientImpl) applySubscription() (err error) {
	m.mux.Lock()
	defer m.mux.Unlock()

	slog.Info("applySubscriptions. Re-applying subscriptions",
		slog.Int("subscriptions", len(m.subscriptions)))
	for k, req := range m.subscriptions {
		_ = k
		_, err = m.ForwardRequestWait(req)
		if err != nil {
			break
		}
	}
	return err
}

func (m *ReconnectClientImpl) AuthenticateWithForm(tdoc *td.TD, getcred api.GetCredentials) error {
	return m.conn.AuthenticateWithForm(tdoc, getcred)
}

// Connect periodically tries a reconnect until successful or the context is cancelled
// This uses an increasing backoff period up to 15 seconds, starting at 1msec.
func (m *ReconnectClientImpl) Connect(ctx context.Context) error {

	var backoffDuration time.Duration = time.Millisecond

	for i := 0; m.maxReconnectAttempts == 0 || i < m.maxReconnectAttempts; i++ {

		// wait the backoff period or until the main context is cancelled before trying again
		sleep, sleepEndFn := context.WithTimeout(ctx, backoffDuration)
		select {
		case <-ctx.Done():
			sleepEndFn()
			return ctx.Err() // fmt.Errorf("Context cancelled")
		case <-sleep.Done():
			sleepEndFn()
			if m.conn.GetConnectionStatus() == api.StatusConnected {
				return nil
			}
			// request a reconnect with the last known parameters
			slog.Info("Connect; client attempt connection", "client module", m.conn.GetThingID())
			err := m.conn.Connect()
			if err == nil {
				// success,
				err = m.applySubscription()
				return err
			}
			// don't retry if client is refused
			if m.conn.GetConnectionStatus() == api.StatusRefused {
				return err
			}
			// backoffDuration += time.Duration(rand.Uint64N(uint64(time.Second)))
			backoffDuration += time.Second
			slog.Info("Connect; reconnect failed. Retrying after backoff...", "backoff", backoffDuration)
		}
	}
	return fmt.Errorf("Unable to reconnect after '%d' attempts", m.maxReconnectAttempts)
}

// Start the reconnect attempt
// This sets the cancelFn so the Close method can interrupt the reconnect
func (m *ReconnectClientImpl) DoReconnect() {
	ctx, cancelFn := context.WithCancel(context.Background())
	m.mux.Lock()
	m.cancelFn = cancelFn
	m.mux.Unlock()

	err := m.Connect(ctx)
	if err != nil {
		slog.Warn("Reconnect failed", "err", err.Error())
	}
	m.mux.Lock()
	cancelFn()
	m.cancelFn = nil
	m.mux.Unlock()

}

func (m *ReconnectClientImpl) GetConnectionStatus() api.ConnectionStatus {
	return m.conn.GetConnectionStatus()
}

// handleConnectChange handles a disconnection callback
// if no reconnect is in progress then start it.
func (m *ReconnectClientImpl) handleConnectChange(
	newStatus api.ConnectionStatus, c api.ITransportClient) {

	// if connection is lost then initiate the reconnect process.
	// note that closing a client can still cause a lost callback, but in that case
	// it should be ignored.
	status := m.conn.GetConnectionStatus()
	if status == api.StatusLost {
		m.mux.Lock()
		defer m.mux.Unlock()
		// only start reconnecting if not already reconnecting
		if m.cancelFn == nil {
			go m.DoReconnect()
		}
	}
}

// Experimental: If no client is linked then monitor the notification for a disconnect
// and send a reconnect request.
func (m *ReconnectClientImpl) HandleNotification(notif *msg.NotificationMessage) {

	if m.conn == nil {
		if notif.AffordanceType == msg.AffordanceTypeEvent &&
			notif.Name == api.ClientConnectionStatusEvent &&
			notif.Data.(api.ConnectionStatus) == api.StatusLost {

			// Send a connect request
			req := msg.NewRequestMessage(
				td.OpInvokeAction, notif.SenderID, api.ClientConnectAction, nil)
			go m.ForwardRequest(req, nil)
		}
	}
	m.HiveModuleBase.HandleNotification(notif)
}

// HandleRequest tracks subscriptions to events and property updates
func (m *ReconnectClientImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	switch req.Operation {
	case td.OpSubscribeAllEvents, td.OpSubscribeEvent,
		td.OpObserveAllProperties, td.OpObserveMultipleProperties, td.OpObserveProperty:

		// TBD: this doesn't differentiate between event/property affordance or single or multiple
		key := fmt.Sprintf("%s-%s-%s", req.Operation, req.ThingID, req.Name)
		m.subscriptions[key] = req

	case td.OpUnobserveAllProperties, td.OpUnobserveMultipleProperties, td.OpUnobserveProperty,
		td.OpUnsubscribeAllEvents, td.OpUnsubscribeEvent:
		// remove the recorded subscription request
		// FIXME: map the unsubscribe/unobserve to the stored operation
		key := fmt.Sprintf("%s-%s-%s", req.Operation, req.ThingID, req.Name)
		delete(m.subscriptions, key)
	}
	// forward
	return m.HiveModuleBase.HandleRequest(req, replyTo)
}

// Start the reconnect module
// If no transport client was provided on startup then see if the request sink is one.
func (m *ReconnectClientImpl) Start() error {
	if m.conn != nil {
		// A failure to connect is not a failure of this module
		// TBD - should this run DoReconnect instead?
		err := m.conn.Start()
		if err != nil {
			slog.Warn("ReconnectClient.Start The linked client failed to start.",
				"err", err.Error(), "client module", m.conn.GetThingID())
		}
	}
	return nil
}
func (m *ReconnectClientImpl) Stop() {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.cancelFn != nil {
		// cancelFn will be cleared when reconnect loop has exited
		m.cancelFn()
	}
	m.conn.Stop()
}

// NewReconnectClientImpl creates a reconnect module for use with the given client.
//
// This module uses the ReconnectModuleType as its ID.
//
//	cl is the transport client connection instance to use before connecting
func NewReconnectClientImpl(cl api.ITransportClient) (m *ReconnectClientImpl) {

	m = &ReconnectClientImpl{
		HiveModuleBase: modules.NewHiveModuleBase(reconnect.ReconnectModuleType, 0),

		maxBackoffTimeLimit: reconnect.DefaultBackoffLimit,

		conn:                 cl,
		maxReconnectAttempts: reconnect.DefaultMaxReconnectAttempts,
		subscriptions:        make(map[string]*msg.RequestMessage),
	}
	// enable the reconnect using the callback
	cl.SetConnectHandler(m.handleConnectChange)
	// link between client and this module
	m.SetRequestSink(m.conn)
	cl.SetNotificationSink(m)

	return m
}
