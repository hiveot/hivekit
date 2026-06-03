package internal

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
	"github.com/hiveot/hivekit/go/modules/transport"
)

// ConnectWSS establishes a websocket session with the server
// This will create a new http-1 client as gorilla websockets does not work with http/2
func ConnectWSS(
	clientID string,
	hostPort string,
	wssPath string,
	cid string,
	bearerToken string,
	clientCert *tls.Certificate,
	caCert *x509.Certificate,
	onConnect func(transport.ConnectionStatus, error),
	onMessage func(raw []byte),
) (cancelFn func(), conn *websocket.Conn, status transport.ConnectionStatus, err error) {

	var clientCertList []tls.Certificate

	slog.Info("ConnectWSS - establishing Websocket connection to server",
		slog.String("server", hostPort),
		slog.String("path", wssPath),
		slog.String("clientID", clientID))

	// each connection a unique cid
	connectURL := "wss://" + hostPort + wssPath
	serverName := strings.Split(hostPort, ":")[0]

	// use context to disconnect the client
	wssCtx, wssCancelFn := context.WithCancel(context.Background())
	onConnect(transport.StatusConnecting, nil)

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
	if bearerToken != "" {
		wssHeader.Add("Authorization", "bearer "+bearerToken)
	}
	wssHeader.Add(transport.ConnectionIDHeader, cid)
	//parts, _ := url.Parse(hostPort)
	//origin := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	//opts.HTTPHeader.Add("Origin", origin)

	wssdialer := *websocket.DefaultDialer // run a copy
	wssdialer.TLSClientConfig = tlsConfig
	wssdialer.NetDialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
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
	// websockets do not support http/2 so stick to http 1.1.
	wssConn, r, err := wssdialer.Dial(connectURL, wssHeader)
	if err != nil {
		status := transport.StatusLost
		if r != nil && r.StatusCode == http.StatusUnauthorized {
			err = fmt.Errorf("ConnectWSS: Connection as '%s' to '%s' failed: %s",
				clientID, connectURL, err.Error())
			slog.Warn(err.Error())
			status = transport.StatusRefused
		}
		wssCancelFn()
		return nil, nil, status, err
	}

	closeWSSFn := func() {
		err = wssConn.Close()

		// is this needed after close above?
		wssCancelFn()
	}
	// notify the world we're connected
	onConnect(transport.StatusConnected, nil)

	// last, start handling incoming messages
	go func() {
		WSSReadLoop(wssCtx, wssConn, onMessage)
		// end of read loop
		onConnect(transport.StatusLost, nil)
	}()

	return closeWSSFn, wssConn, transport.StatusConnected, nil
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
