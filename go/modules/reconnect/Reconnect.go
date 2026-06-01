package reconnect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/utils"
)

const ReconnectModuleType = "reconnect"
const DefaultMaxReconnectAttempts = 999999

// Reconnect is a module that automatically re-applies request a reconnect after a
// client loses its connection and applies event subscriptions and property observations
// after a connection is restored.
type Reconnect struct {
	modules.HiveModuleBase

	// cancel any reconnect attempts.
	cancelFn func()

	//
	maxReconnectAttempts int // 0 for indefinite

	// mutex to block subscription updates
	mux sync.RWMutex

	// record of subscriptions by key="{thingID}-{name}"
	subscriptions map[string]*msg.RequestMessage
}

// applySubscription applies recorded subscriptions
// this will lock subscriptions until complete or error
func (m *Reconnect) applySubscription() (err error) {
	m.mux.Lock()
	defer m.mux.Unlock()

	slog.Info("applySubscriptions. Resubscribing")
	for k, req := range m.subscriptions {
		_ = k
		_, err = m.ForwardRequestWait(req)
		if err != nil {
			break
		}
	}
	return err
}

// DoReconnect periodically tries a reconnect until successful or the context is cancelled
func (m *Reconnect) DoReconnect(ctx context.Context, clientModuleID string) error {

	// This uses an increasing backoff period up to 15 seconds, starting random between 0-2 seconds
	var backoffDuration time.Duration = time.Duration(rand.Uint64N(uint64(time.Second * 2)))

	// pass a reconnect action request to the client module
	req := msg.NewRequestMessage(
		td.OpInvokeAction, clientModuleID, transport.ClientConnectAction, nil)

	for i := 0; m.maxReconnectAttempts == 0 || i < m.maxReconnectAttempts; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err() // fmt.Errorf("Context cancelled")
		default:
			// request a reconnect with the last known parameters
			resp, err := m.ForwardRequestWait(req)
			if err != nil {
				// the request is not supported
				// slog.Warn("HandleNotification: Reconnect not supported", "err", err.Error())
				return err
			}
			// if the reconnect succeeds then re-apply the subscription and return
			if resp.Status == msg.StatusCompleted {
				err = m.applySubscription()
				return err
			}
			// don't retry if client is unauthorized
			if errors.Is(resp.AsError(), utils.UnauthorizedError) {
				return err
			}

			// wait the backoff period or until the main context is cancelled before trying again
			sleep, sleepEndFn := context.WithTimeout(ctx, backoffDuration)
			<-sleep.Done()
			sleepEndFn()
		}
	}
	return fmt.Errorf("Unable to reconnect after '%d' attempts", m.maxReconnectAttempts)
}

// HandleNotification detects a disconnect and reconnect from a client module.
func (m *Reconnect) HandleNotification(notif *msg.NotificationMessage) {

	// if this is a connection lost event then initiate the reconnect process
	// since the client type is unknown, the sender thingID is also unknown.
	// this therefore assumes that the event name is unique (or reserved)
	if notif.AffordanceType == msg.AffordanceTypeEvent &&
		notif.Name == transport.ClientConnectionStatusEvent &&
		notif.Data.(transport.ConnectionStatus) == transport.StatusLost {

		// Start the reconnect process with the client module
		ctx, cancelFn := context.WithCancel(context.Background())
		m.mux.Lock()
		m.cancelFn = cancelFn
		m.mux.Unlock()
		err := m.DoReconnect(ctx, notif.SenderID)
		if err != nil {
			slog.Warn("Reconnect failed", "err", err.Error())
		}
		cancelFn()
	}
	m.HiveModuleBase.HandleNotification(notif)
}

// HandleRequest tracks subscriptions to events and property updates
func (m *Reconnect) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

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

// Start the module
// func (m *Reconnect) Start() error {
// }

// Stop the module and abort any reconnect attempts
// if a reconnect is ongoing it will be stopped before returning.
func (m *Reconnect) Stop() {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.cancelFn != nil {
		m.cancelFn()
		m.cancelFn = nil
	}
}

// NewReconnect returns a new instance of the client auto-reconnect module.
//
//	timeout is the maximum time to wait to reconnect or 0 for the default
func NewReconnect(timeout time.Duration) *Reconnect {
	m := &Reconnect{
		HiveModuleBase:       modules.NewHiveModuleBase(ReconnectModuleType, timeout),
		subscriptions:        make(map[string]*msg.RequestMessage),
		maxReconnectAttempts: DefaultMaxReconnectAttempts,
	}

	return m
}

// Factory for creating a consumer module using the factory environment
func NewReconnectFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	c := NewReconnect(f.GetEnvironment().RpcTimeout)
	return c, nil
}
