package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/teris-io/shortid"
)

type WSSMessage map[string]any

// WssServerConnection is  the server side instance of a connection by a client.
// This implements the IConnection interface for sending messages to
// agent or consumers.
type WssServerConnection struct {
	transports.ServerConnectionBase

	// connection ID
	//connectionID string

	// clientID is the account ID of the connected client
	// clientID string

	// connection request remote address
	httpReq *http.Request

	// isConnected atomic.Bool

	// track last used time to auto-close stale connections
	lastActivity time.Time

	// mutex for controlling writing and closing
	// mux sync.RWMutex

	// converter for request/response messages
	encoder transports.IMessageEncoder

	// notifHandler handles the requests received from the remote producer
	notifHandler msg.NotificationHandler

	// reqHandler handles the requests received from the remote consumer
	reqHandler msg.RequestHandler

	// request-response channel
	// rnrChan *msg.RnRChan

	// how long to wait for a response after sending a request
	// respTimeout time.Duration

	// underlying websocket connection
	wssConn *websocket.Conn
}

// _onMessage handles an incoming websocket message
// The message is converted into a request, response or notification and passed
// on to the registered handler.
func (sc *WssServerConnection) _onMessage(raw []byte) {
	sc.Mux.Lock()
	sc.lastActivity = time.Now()
	sc.Mux.Unlock()
	var notif *msg.NotificationMessage
	var req *msg.RequestMessage
	var resp *msg.ResponseMessage

	// the only way to know which message type it is is to decode it
	notif, err := sc.encoder.DecodeNotification(raw)
	if err == nil {
		notif.SenderID = sc.GetClientID()
		sc.OnNotification(notif, sc.notifHandler)
		return
	}
	resp, err = sc.encoder.DecodeResponse(raw)
	if err == nil {
		// the response hadler is already provided with the request
		resp.SenderID = sc.GetClientID()
		sc.OnResponse(resp)
		return
	}
	req, err = sc.encoder.DecodeRequest(raw)
	if err == nil {
		req.SenderID = sc.GetClientID()
		sc.OnRequest(req, sc.reqHandler)
		return
	}
	slog.Warn("onMessage: Message is not a notification, request or response")
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
		slog.Debug("WssServerConnection.ReadLoop: Remote client disconnected")
		// close channel when no-one is writing
		// in the meantime keep reading to prevent deadlock
		_ = wssConn.Close()

	}()
	// read messages from the client until the connection closes
	for sc.IsConnected() { // sseMsg := range sseChan {
		_, raw, err := wssConn.ReadMessage()
		if err != nil {
			// ending the read loop and returning will close the connection
			break
		}
		// process the message in the background to allow concurrent messages
		go sc._onMessage(raw)
	}
}

// NewWSSServerConnection creates a new Websocket connection instance for use by
// agents and consumers.
// This implements the IConnection interface.
//
// clientID is the consumer or agent authenticated ID
// r is the request used to establish this connection
// wssConn is the connection on which to send/receive messages to the client
// messageConverter maps protocol messages to standard RRN
// reqHandler will handle incoming request messages (required)
// notifHandler will handling incoming notification messages (required)
func NewWSSServerConnection(
	clientID string,
	r *http.Request,
	wssConn *websocket.Conn,
	encoder transports.IMessageEncoder,
	reqHandler msg.RequestHandler,
	notifHandler msg.NotificationHandler,
	// respTimeout time.Duration,
) *WssServerConnection {

	cid := "WSS" + shortid.MustGenerate()
	if reqHandler == nil || notifHandler == nil {
		panic("WSS incoming connection needs request and notification handlers.")
	}

	c := &WssServerConnection{
		wssConn:      wssConn,
		encoder:      encoder,
		httpReq:      r,
		lastActivity: time.Time{},
		// rnrChan:          msg.NewRnRChan(),
		// respTimeout:      respTimeout,
		reqHandler:   reqHandler,
		notifHandler: notifHandler,
	}
	c.Init(clientID, r.URL.String(), cid, encoder, c._sendRaw)

	var _ transports.IConnection = c // interface check
	return c
}
