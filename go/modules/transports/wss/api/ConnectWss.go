package wssapi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httptransport"
	"github.com/teris-io/shortid"
)

// ConnectWSS establishes a websocket session with the server
// This will create a new http-1 client as gorilla websockets does not work with http/2
func ConnectWSS(
	clientID string,
	hostPort string,
	wssPath string,
	bearerToken string,
	clientCert *tls.Certificate,
	caCert *x509.Certificate,
	onConnect func(bool, error),
	onMessage func(raw []byte),
) (cancelFn func(), conn *websocket.Conn, err error) {

	var clientCertList []tls.Certificate

	slog.Info("ConnectWSS (to hub) - establishing Websocket connection to server",
		slog.String("server", hostPort),
		slog.String("path", wssPath),
		slog.String("clientID", clientID))

	// each connection a unique cid
	connectionID := "wss-" + shortid.MustGenerate()
	connectURL := "wss://" + hostPort + wssPath
	serverName := strings.Split(hostPort, ":")[0]

	// use context to disconnect the client
	wssCtx, wssCancelFn := context.WithCancel(context.Background())

	caCertPool := x509.NewCertPool()
	if caCert != nil {
		caCertPool.AddCert(caCert)
	}
	// support for client certificate
	if clientCert != nil {
		clientCertList = []tls.Certificate{*clientCert}
	}
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
		// ServerName is required when InsecureSkipVerify disabled
		ServerName:         serverName,
		InsecureSkipVerify: caCert == nil,
		Certificates:       clientCertList,
	}

	wssHeader := http.Header{}
	wssHeader.Add("Authorization", "bearer "+bearerToken)
	wssHeader.Add(httptransport.ConnectionIDHeader, connectionID)
	//parts, _ := url.Parse(hostPort)
	//origin := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	//opts.HTTPHeader.Add("Origin", origin)

	dialer := *websocket.DefaultDialer // run a copy
	dialer.TLSClientConfig = tlsConfig
	dialer.NetDialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
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
	// FIXME: use http/2
	//httpParts, _ := url.Parse(wssURL)
	//httpParts.Scheme = "https"
	//httpsURL := httpParts.String()

	wssConn, r, err := dialer.Dial(connectURL, wssHeader)
	if err != nil {
		// FIXME: when unauthorized, don't retry. A new token is needed. (session ended).
		if r != nil && r.StatusCode == http.StatusUnauthorized {
			msg := fmt.Sprintf("Unauthorized: Connection as '%s' to '%s' failed: %s",
				clientID, connectURL, err.Error())
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

// WSSReadLoop reads incoming websocket messages in a loop, until connection closes or context is cancelled
func WSSReadLoop(ctx context.Context,
	wssConn *websocket.Conn, onMessage func(raw []byte)) {

	var readLoop atomic.Bool
	readLoop.Store(true)

	// close the client when the context ends drops
	go func() {
		select {
		case <-ctx.Done(): // remote client connection closed
			slog.Debug("WSSReadLoop: Remote client disconnected")
			// close channel when no-one is writing
			// in the meantime keep reading to prevent deadlock
			_ = wssConn.Close()
			readLoop.Store(false)
		}
	}()

	// read messages from the client until the connection closes
	for readLoop.Load() { // sseMsg := range sseChan {
		_, raw, err := wssConn.ReadMessage()
		if err != nil {
			// avoid further writes
			readLoop.Store(false)
			// ending the read loop and returning will close the connection
			break
		}
		// process in the background
		go onMessage(raw)
	}

}
