package wssapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httptransport"
)

// ConnectWSS establishes a websocket session with the server using the given TLS client
// NOTE: As of Jan 2026 this does not work because gorilla websockets dont support http/2.
func ConnectWSS2(
	tlsClient httptransport.ITlsClient,
	wssPath string,
	onConnect func(bool, error),
	onMessage func(raw []byte),
) (cancelFn func(), conn *websocket.Conn, err error) {

	slog.Info("ConnectWSS (to hub) - establishing Websocket connection to server",
		slog.String("path", wssPath),
		slog.String("clientID", tlsClient.GetClientID()),
	)
	hostPort := tlsClient.GetHostPort()
	connectURL := fmt.Sprintf("wss://%s%s", hostPort, wssPath)

	// use context to disconnect the client
	wssCtx, wssCancelFn := context.WithCancel(context.Background())

	// websockets don't work over http/2, so we need a tls.Config for the dialer
	tlsConfig := tlsClient.GetTlsTransport().TLSClientConfig

	wssDialer := *websocket.DefaultDialer // copy the default dialer
	wssDialer.TLSClientConfig = tlsConfig.Clone()
	wssDialer.NetDialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		netConn, err := net.Dial(network, addr)
		if err != nil {
			return nil, err
		}

		// 'NetDialTLSContext' also gets called during the proxy CONNECT for some reason (at this point 'network' equals "TCP" and 'addr' equals "127.0.0.1:8888")
		// The HTTP proxy doesn't support HTTPS however, so I return the established TCP connection early.
		// If I don't do this check, the connection hangs forever (tested with several proxies).
		// This feels kinda hacky though, not sure if this is the correct approach...
		//if p.Host == addr {
		//	return netConn, err
		//}

		// Example TLS handshake
		tlsConn := tls.Client(netConn, tlsConfig)
		if err = tlsConn.Handshake(); err != nil {
			return nil, err
		}

		return tlsConn, nil
	}

	wssConn, r, err := wssDialer.Dial(connectURL, nil)
	if err != nil {
		// FIXME: when unauthorized, don't retry. A new token is needed. (session ended).
		if r != nil && r.StatusCode == http.StatusUnauthorized {
			msg := fmt.Sprintf("Unauthorized: Connection as '%s' to '%s' failed: %s",
				tlsClient.GetClientID(), connectURL, err.Error())
			slog.Warn(msg)
			err = transports.UnauthorizedError
		}
		wssCancelFn()
		return nil, nil, err
	}

	closeWSSFn := func() {
		err = wssConn.Close()

		// is this needed after close above?
		wssCancelFn()
	}
	// notify the world we're connected
	if onConnect != nil {
		onConnect(true, nil)
	}
	// last, start handling incoming messages
	go func() {
		WSSReadLoop(wssCtx, wssConn, onMessage)
		if onConnect != nil {
			onConnect(false, nil)
		}
	}()

	return closeWSSFn, wssConn, nil
}
