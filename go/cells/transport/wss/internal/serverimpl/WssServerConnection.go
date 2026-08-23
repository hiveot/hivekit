package serverimpl

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells/transport"
	"github.com/teris-io/shortid"
)

type WSSMessage map[string]any

// WssServerConnection is  the server side instance of a connection by a client.
// This implements the IConnection interface for sending messages to
// devices, services or consumers.
type WssServerConnection struct {
	// ServerConnectionBase handles the generic messaging part
	*transport.ServerConnectionBase

	// connection request remote address
	httpReq *http.Request

	// converter for request/response messages
	encoder transport.IMessageEncoder

	// notifHandler handles the requests received from the remote producer
	notifHandler msg.NotificationHandler

	// reqHandler handles the requests received from the remote consumer
	reqHandler msg.RequestHandler

	// underlying websocket connection
	wssConn *websocket.Conn
}

// _sendRaw sends the seriaziled websocket message to the connected client
// msgType is ignored since the socket doesn't support metadata
func (sc *WssServerConnection) _sendRaw(msgType string, msg []byte) (err error) {

	if !sc.IsConnected() {
		err = fmt.Errorf(
			"_send: connection with client '%s' is now closed", sc.GetClientID())
		slog.Warn(err.Error())
	} else {
		// websockets do not allow concurrent write
		sc.Mux.Lock()
		defer sc.Mux.Unlock()
		err = sc.wssConn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			err = fmt.Errorf("WssServerConnection._send write error: %s", err)
		}
	}
	return err
}

// Close closes the connection and ends the read loop
func (sc *WssServerConnection) Close() {
	sc.Mux.Lock()
	defer sc.Mux.Unlock()
	if sc.IsConnected() {
		_ = sc.wssConn.Close()
	}
}

// ReadLoop reads incoming websocket messages in a loop, until connection closes or context is cancelled
func (sc *WssServerConnection) ReadLoop(ctx context.Context, wssConn *websocket.Conn) {

	// close the client when the context ends drops
	go func() {
		<-ctx.Done() // remote client connection closed
		slog.Debug("WssServerConnection.ReadLoop: context cancelled")
		// close channel when no-one is writing
		// in the meantime keep reading to prevent deadlock
		_ = wssConn.Close()

	}()
	// read messages from the client until the connection closes
	for sc.IsConnected() { // sseMsg := range sseChan {
		_, raw, err := wssConn.ReadMessage()
		if err != nil {
			slog.Info("WssServerConnection.ReadLoop: Remote client disconnected")
			// ending the read loop and returning will close the connection
			break
		}
		// process the message in the background to allow concurrent messages
		// all the logic is generic, so handled in the server base
		go sc.OnRemoteMessage(raw)
	}
}

// NewWSSServerConnection creates a new Websocket connection instance for use by
// devices, services and consumers.
//
// This implements the IConnection interface. Use Close() to close the
// connection from the server end.
//
// clientID is the consumer, device or service authenticated ID
// r is the request used to establish this connection
// wssConn is the connection on which to send/receive messages to the client
// messageConverter maps protocol messages to standard RRN
// reqHandler will handle incoming request messages (required)
// notifHandler will handling incoming notification messages (required)
func NewWSSServerConnection(
	clientID string,
	r *http.Request,
	wssConn *websocket.Conn,
	encoder transport.IMessageEncoder,
	reqHandler msg.RequestHandler,
	notifHandler msg.NotificationHandler,
) *WssServerConnection {

	connectionID := "WSS" + shortid.MustGenerate()
	if reqHandler == nil || notifHandler == nil {
		panic("WSS incoming connection needs request and notification handlers.")
	}

	c := &WssServerConnection{
		wssConn:      wssConn,
		encoder:      encoder,
		httpReq:      r,
		reqHandler:   reqHandler,
		notifHandler: notifHandler,
	}
	remoteAddr := r.URL.String()
	c.ServerConnectionBase = transport.NewServerConnectionBase(
		clientID, remoteAddr, connectionID,
		encoder, c._sendRaw, reqHandler, notifHandler,
	)

	var _ api.IConnection = c // interface check
	return c
}
