package ssescclient

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/ssesc"
	"github.com/hiveot/hivekit/go/msg"
	sse "github.com/tmaxmax/go-sse"
)

// maxSSEMessageSize allow this maximum size of an SSE message
// go-sse allows this increase of allocation size on receiving messages
const maxSSEMessageSize = 1024 * 1024 * 10

// ConnectSSE establishes a new sse connection using the given http client.
// This provides the input channel for notifications, requests, and responses.
//
// If the connection is interrupted, the sse connection retries with backoff period.
// If an authentication error occurs then the onDisconnect handler is invoked with an error.
// If the connection is cancelled then the onDisconnect is invoked without error
//
// This invokes onConnect when the connection is lost. The caller must handle the
// connection established when the first ping is received after successful connection.
func ConnectSSE(
	tlsClient transports.ITlsClient, // TLS client with bearer token
	ssePath string,
	onConnect func(bool, error),
	onMessage func(event sse.Event),
	timeout time.Duration,
) (cancelFn func(), err error) {

	method := http.MethodGet
	body := []byte{}
	contentType := "application/JSON"

	sseCtx, sseCancelFn := context.WithCancel(context.Background())

	clientID := tlsClient.GetClientID()

	// replace the sse:// schema with https:// required for the request itself

	r := tlsClient.CreateRequest(sseCtx, method, ssePath, nil, body, contentType)
	sseClient := &sse.Client{
		HTTPClient: tlsClient.GetHttpClient(),
		OnRetry: func(err error, backoff time.Duration) {
			slog.Warn("SSE Connection retry",
				"err", err, "sse path", ssePath,
				"backoff", backoff)
			// TODO: how to be notified if the connection is restored?
			//  workaround: in handleSSEEvent, update the connection status
			onConnect(false, err)
		},
	}
	conn := sseClient.NewConnection(r)

	// increase the maximum buffer size to 1M (_maxSSEMessageSize)
	// note this requires go-sse v0.9.0-pre.2 as a minimum.
	//https://github.com/tmaxmax/go-sse/issues/32
	newBuf := make([]byte, 0, 1024*65)
	// TODO: make limit configurable
	conn.Buffer(newBuf, maxSSEMessageSize)
	remover := conn.SubscribeToAll(onMessage)

	// Wait for max 3 seconds to detect a connection
	waitConnectCtx, waitConnectCancelFn := context.WithTimeout(context.Background(), timeout)
	conn.SubscribeEvent(ssesc.SSEPingEvent, func(event sse.Event) {
		// WORKAROUND since go-sse has no callback for a successful (re)connect, simulate one here.
		// As soon as a connection is established the server could send a 'ping' event.
		// success!
		slog.Info("handleSSEEvent: connection (re)established; setting connected to true")
		onConnect(true, nil)
		waitConnectCancelFn()
	})
	var sseConnErr atomic.Pointer[sse.ConnectionError]
	go func() {
		// connect and wait until the connection ends
		// and report an error if connection ends due to reason other than context cancelled
		// onConnect will be called on receiving the first (ping) message
		//onConnect(true, nil)
		err := conn.Connect()

		if connError, ok := err.(*sse.ConnectionError); ok {
			if strings.Index(connError.Error(), "401") >= 0 {
				// this is an authentication error
				slog.Error("SSE authentication failed",
					"clientID", clientID,
					"err", err.Error())
			} else {
				// since sse retries, this is likely an authentication error
				slog.Error("SSE connection failed (server shutdown or connection interrupted)",
					"clientID", clientID,
					"err", err.Error())
			}
			sseConnErr.Store(connError)
			//err = fmt.Errorf("connect Failed: %w", connError.Err) //connError.Err
			waitConnectCancelFn()
		} else if errors.Is(err, context.Canceled) {
			// context was cancelled. no error
			err = nil
		}
		onConnect(false, err)
		_ = remover
	}()

	// wait for the SSE connection to be established
	<-waitConnectCtx.Done()
	e := waitConnectCtx.Err()
	if errors.Is(e, context.DeadlineExceeded) {
		err = fmt.Errorf("ConnectSSE: Timeout connecting to the server")
		slog.Warn(err.Error())
		sseCancelFn()
	} else if sseConnErr.Load() != nil {
		err = sseConnErr.Load()
		// something else went wrong
		slog.Warn("ConnectSSE: error" + err.Error())
	}
	closeSSEFn := func() {
		// any other cleanup?
		sseCancelFn()
	}
	return closeSSEFn, err
}

// ConnectSSE establishes the sse connection using the given bearer token
// cl.handleSseEvent will set 'connected' status when the first ping event is
// received from the server. (go-sse doesn't have a connected callback)
func (cl *SseScClient) ConnectSSE(token string) (err error) {
	if cl.ssePath == "" {
		return fmt.Errorf("connectSSE: Missing SSE path")
	}
	// establish the SSE connection for the return channel
	//sseURL := fmt.Sprintf("https://%s%s", cc.hostPort, cc.ssePath)

	cl.sseCancelFn, err = ConnectSSE(
		// use the same http client for both http requests and sse connection
		cl.tlsClient,
		cl.ssePath,
		cl.handleSSEConnect,
		cl.handleSseEvent,
		cl.timeout)

	return err
}

// handler when the SSE connection is established or fails.
// This invokes the connectHandler callback if provided.
func (cl *SseScClient) handleSSEConnect(connected bool, err error) {
	errMsg := ""
	clientID := cl.GetClientID()
	cid := cl.tlsClient.GetConnectionID()

	// if the context is cancelled this is not an error
	if err != nil {
		errMsg = err.Error()
	}
	slog.Info("handleSSEConnect",
		slog.String("clientID", clientID),
		slog.String("connectionID", cid),
		slog.Bool("connected", connected),
		slog.String("err", errMsg))

	var connectionChanged bool = false
	if cl.IsConnected() != connected {
		connectionChanged = true
	}
	cl.SetConnected(connected)
	if err != nil {
		cl.mux.Lock()
		cl.lastError.Store(&err)
		cl.mux.Unlock()
	}

	// Note: this callback can send notifications to the client,
	// so prevent deadlock by running in the background.
	// (caught by readhistory failing for unknown reason)
	if connectionChanged && cl.connectHandler != nil {
		go func() {
			cl.connectHandler(connected, cl, err)
		}()
	}
}

// handleSSEEvent processes the push-event received from the server.
// This splits the message into notification, response and request
// requests have an operation and correlationID
// responses have no operations and a correlationID
// notifications have an operations and no correlationID
func (cl *SseScClient) handleSseEvent(event sse.Event) {
	clientID := cl.tlsClient.GetClientID()

	// no further processing of a ping needed
	if event.Type == ssesc.SSEPingEvent {
		return
	}

	// Use the hiveot message envelopes for request, response and notification
	switch event.Type {
	case msg.MessageTypeNotification:
		notif := cl.msgConverter.DecodeNotification([]byte(event.Data))
		if notif == nil {
			return
		}
		if cl.requestSink == nil {
			slog.Error("HandleSseEvent: no sink set. Notification is dropped.",
				"clientID", clientID,
				"operation", notif.Operation,
				"name", notif.Name,
			)
		} else if cl.notificationSink != nil {
			// notifications received from the server are passed to the registered handler
			go func() {
				cl.notificationSink(notif)
			}()
		} else {
			// notifications are only received when subscribed so someone forgot to
			// set a handler.
			slog.Error("handleSseEvent: Received notification but no handler is set")
		}
	case msg.MessageTypeRequest:
		var err error
		req := cl.msgConverter.DecodeRequest([]byte(event.Data))
		if req == nil {
			return
		}
		if cl.requestSink == nil {
			err = fmt.Errorf("handleSseEvent: no requestSink set. Request is dropped.")
			slog.Error("handleSseEvent: no sink set. Request is dropped.",
				"clientID", clientID,
				"operation", req.Operation,
				"name", req.Name,
				"senderID", req.SenderID,
			)
		} else {
			err = cl.requestSink(req, func(resp *msg.ResponseMessage) error {
				// return the response to the caller
				err2 := cl.SendResponse(resp)
				return err2
			})
			// an error means the request could not be handled
		}
		// responses are optional
		if err != nil {
			resp := req.CreateErrorResponse(err)
			_ = cl.SendResponse(resp)
		}

	case msg.MessageTypeResponse:
		resp := cl.msgConverter.DecodeResponse([]byte(event.Data))
		if resp == nil {
			slog.Info("handleSseEvent: Received SSE Event but decoder returns nil", "data", string(event.Data))
			return
		}

		// consumer receives a response
		// this will be 'handled' if it was waiting on its rnr channel
		handled := cl.rnrChan.HandleResponse(resp)

		if !handled {
			slog.Warn("handleSseEvent: No response handler for request, response is lost",
				"correlationID", resp.CorrelationID,
				"op", resp.Operation,
				"thingID", resp.ThingID,
				"name", resp.Name,
				"clientID", clientID,
			)
		} else {
			// slog.Info("SSE Response was handled in RnR",
			// "op", resp.Operation, "correlationID", resp.CorrelationID)
		}
	default:
		if cl.notificationSink == nil {
			slog.Error("handleSseEvent, received unexpected message",
				"messageType", event.Type)
			return
		}

		// all other events are intended for other use-cases such as the UI,
		// and can have a formats of event/{dThingID}/{name}
		// Attempt to deliver this for compatibility with other protocols (such has hiveoview test client)
		notif := msg.NotificationMessage{}
		notif.MessageType = msg.MessageTypeNotification
		notif.Value = event.Data
		notif.Operation = event.Type
		// don't block the receiver flow
		go func() {
			cl.notificationSink(&notif)
		}()
	}
}
