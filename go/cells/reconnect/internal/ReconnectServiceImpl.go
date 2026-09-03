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
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/reconnect"
)

// ReconnectServiceImpl is a service that automatically reconnects a transport client when
// it loses its connection, and restores event and property subscriptions.
//
// If a connection fails repeatedly a backoff time is increased until the set limit.
//
// The transport client must be provided on instantiation.
//
// TBD: instead of providing a transport client can the next service in the request chain
// be used instead?.  This is a use-case for obtaining a downstream cell of a type.
type ReconnectServiceImpl struct {
	*cells.HiveCellBase

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
func (svc *ReconnectServiceImpl) applySubscription() (err error) {
	svc.mux.Lock()
	defer svc.mux.Unlock()

	slog.Info("applySubscriptions. Re-applying subscriptions",
		slog.Int("subscriptions", len(svc.subscriptions)))
	for k, req := range svc.subscriptions {
		_ = k
		_, err = svc.EmitRequestWait(req)
		if err != nil {
			break
		}
	}
	return err
}

// func (m *ReconnectServiceImpl) AuthenticateWithTD(tdoc *td.TD, getcred api.GetCredentials) error {
// 	return m.conn.AuthenticateWithTD(tdoc, getcred)
// }

// Connect periodically tries a reconnect until successful or the context is cancelled
// This uses an increasing backoff period up to 15 seconds, starting at 1msec.
func (svc *ReconnectServiceImpl) Connect(ctx context.Context) error {

	var backoffDuration time.Duration = time.Millisecond

	for i := 0; svc.maxReconnectAttempts == 0 || i < svc.maxReconnectAttempts; i++ {

		// wait the backoff period or until the main context is cancelled before trying again
		sleep, sleepEndFn := context.WithTimeout(ctx, backoffDuration)
		select {
		case <-ctx.Done():
			sleepEndFn()
			return ctx.Err() // fmt.Errorf("Context cancelled")
		case <-sleep.Done():
			sleepEndFn()
			if svc.conn.GetConnectionStatus() == api.StatusConnected {
				return nil
			}
			// request a reconnect with the last known parameters
			slog.Info("Connect; client attempt connection", "client cell", svc.conn.GetThingID())
			err := svc.conn.Connect()
			if err == nil {
				// success,
				err = svc.applySubscription()
				return err
			}
			// don't retry if client is refused
			if svc.conn.GetConnectionStatus() == api.StatusRefused {
				return err
			}
			// backoffDuration += time.Duration(rand.Uint64N(uint64(time.Second)))
			backoffDuration += time.Second
			slog.Info("Connect; reconnect failed. Retrying after backoff...", "backoff", backoffDuration)
		}
	}
	return fmt.Errorf("Unable to reconnect after '%d' attempts", svc.maxReconnectAttempts)
}

// Start the reconnect attempt
// This sets the cancelFn so the Close method can interrupt the reconnect
func (svc *ReconnectServiceImpl) DoReconnect() {
	ctx, cancelFn := context.WithCancel(context.Background())
	svc.mux.Lock()
	svc.cancelFn = cancelFn
	svc.mux.Unlock()

	err := svc.Connect(ctx)
	if err != nil {
		slog.Warn("Reconnect failed", "err", err.Error())
	}
	svc.mux.Lock()
	cancelFn()
	svc.cancelFn = nil
	svc.mux.Unlock()

}

func (svc *ReconnectServiceImpl) GetConnectionStatus() api.ConnectionStatus {
	return svc.conn.GetConnectionStatus()
}

// handleConnectChange handles a disconnection callback
// if no reconnect is in progress then start it.
func (svc *ReconnectServiceImpl) handleConnectChange(
	newStatus api.ConnectionStatus, c api.ITransportClient) {

	// if connection is lost then initiate the reconnect process.
	// note that closing a client can still cause a lost callback, but in that case
	// it should be ignored.
	status := svc.conn.GetConnectionStatus()
	if status == api.StatusLost {
		svc.mux.Lock()
		defer svc.mux.Unlock()
		// only start reconnecting if not already reconnecting
		if svc.cancelFn == nil {
			go svc.DoReconnect()
		}
	}
}

// Forward notification received from the connected client.
func (svc *ReconnectServiceImpl) HandleNotification(notif *msg.NotificationMessage) {

	// For now the reconnect is triggered by the callback, not notifications.
	//
	//
	// if svc.conn == nil {
	// 	//  If this is a 'connection lost' event, sent by the client, then send the client a request to
	// 	// reconnect.
	// 	if notif.AffordanceType == msg.AffordanceTypeEvent &&
	// 		notif.Name == api.ClientConnectionStatusEvent &&
	// 		notif.Data.(api.ConnectionStatus) == api.StatusLost {

	// 		// Send a connect request. This uses the notification senderID as the thingID of
	// 		// the client that needs to reconnect.
	// 		req := msg.NewRequestMessage(
	// 			td.OpInvokeAction, notif.SenderID, api.ClientConnectAction, nil)
	// 		go svc.ForwardRequest(req, nil)
	// 	}
	// }
	svc.HiveCellBase.HandleNotification(notif)
}

// HandleRequest tracks subscriptions to events and property updates
func (svc *ReconnectServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	switch req.Operation {
	case td.OpSubscribeAllEvents, td.OpSubscribeEvent,
		td.OpObserveAllProperties, td.OpObserveMultipleProperties, td.OpObserveProperty:

		// TBD: this doesn't differentiate between event/property affordance or single or multiple
		key := fmt.Sprintf("%s-%s-%s", req.Operation, req.ThingID, req.Name)
		svc.subscriptions[key] = req

	case td.OpUnobserveAllProperties, td.OpUnobserveMultipleProperties, td.OpUnobserveProperty,
		td.OpUnsubscribeAllEvents, td.OpUnsubscribeEvent:
		// remove the recorded subscription request
		// FIXME: map the unsubscribe/unobserve to the stored operation
		key := fmt.Sprintf("%s-%s-%s", req.Operation, req.ThingID, req.Name)
		delete(svc.subscriptions, key)
	}
	// forward
	return svc.HiveCellBase.HandleRequest(req, replyTo)
}

// SetRequestSink registers the given sink as the client if one isn't set.
// requestSink must implement the ITransportClient interface so it can be used to
// register the connect callback.
func (svc *ReconnectServiceImpl) SetRequestSink(requestSink api.IHiveCell) {
	svc.HiveCellBase.SetRequestSink(requestSink)

	// attempt to use the sink if no client is set yet
	tpClient, ok := requestSink.(api.ITransportClient)
	if !ok {
		slog.Error("SetRequestSink: not a transport client. Continuing")
		return
	}
	svc.conn = tpClient
	svc.conn.SetConnectHandler(svc.handleConnectChange)

	status := tpClient.GetConnectionStatus()
	if status != api.StatusConnected && status != api.StatusConnecting {

		// FIXME: how to report an authentication failure:
		err := svc.conn.Connect()
		if err != nil {
			slog.Warn("StartReconnectServiceImpl. The linked client failed to start.",
				"err", err.Error(), "client ID", tpClient.GetThingID())
		}
	}
}

// Stop the reconnect service and disconnect the client
func (svc *ReconnectServiceImpl) Stop() {
	svc.mux.Lock()
	defer svc.mux.Unlock()
	if svc.cancelFn != nil {
		// cancelFn will be cleared when reconnect loop has exited
		svc.cancelFn()
	}
	svc.conn.Stop()
}

// StartReconnectServiceImpl creates a reconnect service for use with a transport client.
//
// If a transport client is provided as the sink then make it the request sink for this service so
// it can receive requests, and make this the notification sink for that client so notifications
// are received and forwarded.
//
// The also registers the connection changed callback with the client to receive disconnect
// notifications to trigger reconnect.
// The client must have its authentication. Start will call Connect on this client if not yet
// connected.
//
// This service uses the ReconnectCellType as its ID.
//
//	tpClient is the connected transport client that is a sink for this service.
//	  optional, if not provided SetRequestSink will set the handler.
func StartReconnectServiceImpl(
	tpClient api.ITransportClient) (svc *ReconnectServiceImpl, err error) {

	svc = &ReconnectServiceImpl{
		HiveCellBase: cells.NewHiveCellBase(reconnect.ReconnectCellType, 0),

		maxBackoffTimeLimit: reconnect.DefaultBackoffLimit,

		conn:                 tpClient,
		maxReconnectAttempts: reconnect.DefaultMaxReconnectAttempts,
		subscriptions:        make(map[string]*msg.RequestMessage),
	}
	// link between transport client and this service, if provided.
	if tpClient != nil {
		// Get ready to receive connection notifications from the client.
		tpClient.SetNotificationSink(svc)
		// SetRequestSink will invoke Connect on the client if neccesary.
		svc.SetRequestSink(tpClient)
	}
	return svc, err
}
