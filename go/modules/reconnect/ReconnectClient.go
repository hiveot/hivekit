package reconnect

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transport"
)

const ReconnectModuleType = "reconnect"
const DefaultMaxReconnectAttempts = 999999
const DefaultBackoffLimit = time.Minute * 5

// ReconnectClient is a module that automatically reconnects a transport client when
// it loses its connection, and restores event and property subscriptions.
//
// If a connection fails repeatedly a backoff time is increased until the set limit.
//
// The transport client must be provided on instantiation.
//
// TBD: instead of providing a transport client can the next module in the request chain
// be used instead?.  This is a use-case for obtaining a downstream module of a type.
type ReconnectClient struct {
	*modules.HiveModuleBase

	// cancel any reconnect attempts.
	// this is nil if not connecting
	cancelFn func()

	// the client connection instance
	conn transport.ITransportClient
	//
	maxReconnectAttempts int // 0 for indefinite

	// limit to the reconnect backoff period
	maxBackoffTimeLimit time.Duration

	// mutex to block subscription updates
	mux sync.RWMutex

	// record of subscriptions by key="{thingID}-{name}"
	subscriptions map[string]*msg.RequestMessage
}

// applySubscription applies recorded subscriptions
// this will lock subscriptions until complete or error
func (m *ReconnectClient) applySubscription() (err error) {
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

func (m *ReconnectClient) AuthenticateWithForm(tdoc *td.TD, getcred transport.GetCredentials) error {
	return m.conn.AuthenticateWithForm(tdoc, getcred)
}

// Connect periodically tries a reconnect until successful or the context is cancelled
// This uses an increasing backoff period up to 15 seconds, starting at 1msec.
func (m *ReconnectClient) Connect(ctx context.Context) error {

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
			if m.conn.GetConnectionStatus() == transport.StatusConnected {
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
			if m.conn.GetConnectionStatus() == transport.StatusRefused {
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
func (m *ReconnectClient) DoReconnect() {
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

func (m *ReconnectClient) GetConnectionStatus() transport.ConnectionStatus {
	return m.conn.GetConnectionStatus()
}

// handleConnectChange handles a disconnection callback
// if no reconnect is in progress then start it.
func (m *ReconnectClient) handleConnectChange(
	newStatus transport.ConnectionStatus, c transport.ITransportClient) {

	// if connection is lost then initiate the reconnect process.
	// note that closing a client can still cause a lost callback, but in that case
	// it should be ignored.
	status := m.conn.GetConnectionStatus()
	if status == transport.StatusLost {
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
func (m *ReconnectClient) HandleNotification(notif *msg.NotificationMessage) {

	if m.conn == nil {
		if notif.AffordanceType == msg.AffordanceTypeEvent &&
			notif.Name == transport.ClientConnectionStatusEvent &&
			notif.Data.(transport.ConnectionStatus) == transport.StatusLost {

			// Send a connect request
			req := msg.NewRequestMessage(
				td.OpInvokeAction, notif.SenderID, transport.ClientConnectAction, nil)
			go m.ForwardRequest(req, nil)
		}
	}
	m.HiveModuleBase.HandleNotification(notif)
}

// HandleRequest tracks subscriptions to events and property updates
func (m *ReconnectClient) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	switch req.Operation {
	case td.OpSubscribeAllEvents, td.OpSubscribeEvent,
		td.OpObserveAllProperties, td.OpObserveMultipleProperties, td.OpObserveProperty:

		// TBD: this doesn't differentiate between event/property affordance or single or multiple
		// TODO: how to handle subscription to multiple properties?
		key := fmt.Sprintf("%s-%s", req.ThingID, req.Name)
		m.subscriptions[key] = req

	case td.OpUnobserveAllProperties, td.OpUnobserveMultipleProperties, td.OpUnobserveProperty,
		td.OpUnsubscribeAllEvents, td.OpUnsubscribeEvent:
		// remove the recorded subscription request
		// TODO: remove all on a disconnect request
		key := fmt.Sprintf("%s-%s", req.ThingID, req.Name)
		delete(m.subscriptions, key)
	}
	// forward
	return m.HiveModuleBase.HandleRequest(req, replyTo)
}

// Start the reconnect module
// If no transport client was provided on startup then see if the request sink is one.
func (m *ReconnectClient) Start() error {
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
func (m *ReconnectClient) Stop() {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.cancelFn != nil {
		// cancelFn will be cleared when reconnect loop has exited
		m.cancelFn()
	}
	m.conn.Stop()
}

// NewReconnectClient creates a reconnect module for use with the given client.
//
//	cl is the transport client connection instance to use before connecting
func NewReconnectClient(cl transport.ITransportClient) (m *ReconnectClient) {

	m = &ReconnectClient{
		HiveModuleBase: modules.NewHiveModuleBase(ReconnectModuleType, 0),

		maxBackoffTimeLimit: DefaultBackoffLimit,

		conn:                 cl,
		maxReconnectAttempts: DefaultMaxReconnectAttempts,
		subscriptions:        make(map[string]*msg.RequestMessage),
	}
	// enable the reconnect using the callback
	cl.SetConnectHandler(m.handleConnectChange)
	// link between client and this module
	m.SetRequestSink(m.conn.HandleRequest)
	cl.SetNotificationSink(m.HandleNotification)

	return m
}

// Factory for creating a consumer module using the factory environment
func NewReconnectFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	// env := f.GetEnvironment()
	// TODO: figure out how to include this in a recipe without knowing what client to use
	c := NewReconnectClient(nil)
	return c, nil
}
