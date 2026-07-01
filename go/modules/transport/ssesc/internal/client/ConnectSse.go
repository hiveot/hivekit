package internal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/transport/ssesc"
	"github.com/hiveot/hivekit/go/modules/transport/tlsclient"
	gosse "github.com/tmaxmax/go-sse"
)

// maxSSEMessageSize allow this maximum size of an SSE message
// go-sse allows this increase of allocation size on receiving messages
const maxSSEMessageSize = 1024 * 1024 * 10

// ConnectSSE establishes a new sse connection using the given http client.
// This provides the input channel for notifications, requests, and responses.
//
// If the connection is interrupted this invokes the onConnect callback.
// Retry is disabled since subscriptions are lost.
func ConnectSSE(
	tlsClient tlsclient.ITLSClient, // TLS client with bearer token
	ssePath string,
	onConnect func(status api.ConnectionStatus, err error),
	onMessage func(event gosse.Event),
	timeout time.Duration,
) (cancelFn func(), err error) {

	method := http.MethodGet
	body := []byte{}
	contentType := "application/JSON"

	sseCtx, sseCancelFn := context.WithCancel(context.Background())

	clientID := tlsClient.GetClientID()
	onConnect(api.StatusConnecting, nil)

	// replace the sse:// schema with https:// required for the request itself
	fullURL := fmt.Sprintf("https://%s%s", tlsClient.GetHostPort(), ssePath)
	r := tlsClient.CreateRequest(sseCtx, method, fullURL, nil, body, contentType)
	sseClient := &gosse.Client{
		HTTPClient: tlsClient.GetHttpClient(),
		// OnRetry: func(err error, backoff time.Duration) {
		// 	slog.Warn("SSE Connection retry",
		// 		"err", err, "sse path", ssePath,
		// 		"backoff", backoff)
		// 	// TODO: how to be notified if the connection is restored?
		// 	//  workaround: in handleSSEEvent, update the connection status
		// 	onConnect(false, err)
		// },
		// do not retry
		Backoff: gosse.Backoff{MaxRetries: -1},
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
	conn.SubscribeEvent(ssesc.SSEPingEvent, func(event gosse.Event) {
		// WORKAROUND since go-sse has no callback for a successful (re)connect, simulate one here.
		// As soon as a connection is established the server could send a 'ping' event.
		// success!
		slog.Info("handleSSEEvent: connection (re)established; setting connected to true")
		onConnect(api.StatusConnected, nil)
		waitConnectCancelFn()
	})
	var sseConnErr atomic.Pointer[gosse.ConnectionError]
	go func() {
		// connect and wait until the connection ends
		// and report an error if connection ends due to reason other than context cancelled
		// onConnect will be called on receiving the first (ping) message
		//onConnect(true, nil)
		err := conn.Connect()
		var status api.ConnectionStatus

		if connError, ok := err.(*gosse.ConnectionError); ok {
			if strings.Contains(connError.Error(), "401") {
				// this is an authentication error
				slog.Error("SSE authentication failed. Invalid credentials.",
					"clientID", clientID,
					"err", err.Error())
				status = api.StatusRefused
			} else {
				slog.Warn("SSE connection failed (server shutdown or connection interrupted)",
					"clientID", clientID,
					"err", err.Error())
				status = api.StatusLost
			}
			sseConnErr.Store(connError)
			//err = fmt.Errorf("connect Failed: %w", connError.Err) //connError.Err
			waitConnectCancelFn()
		} else if errors.Is(err, context.Canceled) {
			// context was closed. no error
			err = nil
			status = api.StatusClosed
		}
		onConnect(status, err)
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
