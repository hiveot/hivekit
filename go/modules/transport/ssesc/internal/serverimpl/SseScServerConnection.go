package serverimpl

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/ssesc"
)

type SSEEvent struct {
	EventType string // type of message, e.g. event, action or other
	Payload   string // message content
}

// SseScServerConnection handles the SSE connection by remote client
//
// The Sse-sc (sse single connection) protocol binding uses a 'hiveot' message
// envelope for sending messages between server and consumer.
//
// The sse server connection is a 1-way connection intended for sending messages
// to a client that connects over SSE. The client will use http to send messages
// to the server.
//
// This implements the IConnection interface for sending messages to the client over SSE.
type SseScServerConnection struct {
	// ServerConnectionBase handles the generic messaging part with RnR and timeouts
	*transport.ServerConnectionBase

	// connection remote address
	remoteAddr string

	// incoming sse request
	httpReq *http.Request

	// track last used time to auto-close inactive cm
	lastActivity time.Time

	sseChan chan SSEEvent
}

// _send sends a request, response or notification message to the client over SSE.
// This is different from the WoT SSE subprotocol in that the payload is the
// message envelope and can carry any operation.
func (sc *SseScServerConnection) _sendRaw(msgType string, raw []byte) (err error) {

	sseMsg := SSEEvent{
		EventType: msgType,
		Payload:   string(raw),
	}
	sc.Mux.Lock()
	defer sc.Mux.Unlock()
	if sc.IsConnected() {
		slog.Debug("_send",
			slog.String("to", sc.GetClientID()),
			slog.String("MessageType", msgType),
		)
		sc.sseChan <- sseMsg
	} else {
		slog.Error("HiveotSseServerConnection unable to send message. Connection lost.",
			"msgType", msgType, "clientID", sc.ClientID)
	}
	// as long as the channel exists, delivery will take place
	return nil
}

// Close closes the connection and ends the read loop
func (sc *SseScServerConnection) Close() {
	sc.Mux.Lock()
	defer sc.Mux.Unlock()
	if sc.IsConnected() {
		if sc.sseChan != nil {
			close(sc.sseChan)
		}
		sc.ServerConnectionBase.Disconnect()
	}
}

// Serve serves SSE return channel to the client.
//
// This listens for outgoing requests on the given channel
// It ends when the client disconnects or the connection is closed with Close()
// Sse requests are refused if no valid session is found.
func (sc *SseScServerConnection) Serve(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE response
	//w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate, max-age=0, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Content-Encoding", "none") //https://stackoverflow.com/questions/76375157/client-not-receiving-server-sent-events-from-express-js-server

	// establish a client event channel for sending messages back to the client
	sc.Mux.Lock()
	sc.sseChan = make(chan SSEEvent, 1)
	sc.Mux.Unlock()

	// _send a ping event as the go-sse client doesn't have a 'connected callback'
	pingEvent := SSEEvent{EventType: ssesc.SSEPingEvent}
	sc.Mux.Lock()
	sc.sseChan <- pingEvent
	sc.Mux.Unlock()

	slog.Debug("SseConnection.Serve new SSE connection",
		slog.String("clientID", sc.GetClientID()),
		slog.String("connectionID", sc.ConnectionID),
		slog.String("protocol", r.Proto),
		slog.String("remoteAddr", sc.remoteAddr),
	)
	sendLoop := true

	// close the channel when the connection drops
	go func() {

		<-r.Context().Done() // remote client connection closed

		slog.Debug("SseConnection: Remote client disconnected (read context)")
		// close channel when no-one is writing
		// in the meantime keep reading to prevent deadlock
		sc.Disconnect()

	}()

	// read the message channel for sending messages until it closes
	for sendLoop { // sseMsg := range sseChan {

		// keep reading to prevent blocking on channel on write
		sseMsg, ok := <-sc.sseChan // received event
		var err error

		if !ok { // channel was closed by session
			// avoid further writes
			sendLoop = false
			// ending the read loop and returning will close the connection
			break
		}
		slog.Info("SseConnection: sending sse event to client",
			//slog.String("sessionID", c.sessionID),
			slog.String("clientID", sc.GetClientID()),
			slog.String("connectionID", sc.ConnectionID),
			slog.String("sse eventType", sseMsg.EventType),
		)
		var n int
		n, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n",
			sseMsg.EventType, sseMsg.Payload)
		//_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n",
		//	sseMsg.EventType, sseMsg.ID, sseMsg.Payload)
		if err != nil {
			// the connection might be closing.
			// don't exit the loop until the receive-channel is closed.
			// just keep processing the message until that happens
			// closed go channels panic when written to. So keep reading.
			slog.Error("SseConnection: Error writing SSE event",
				slog.String("Event", sseMsg.EventType),
				slog.String("SenderID", sc.GetClientID()),
				slog.Int("size", len(sseMsg.Payload)),
			)
		} else {
			slog.Debug("SseConnection: SSE write to client",
				slog.String("SenderID", sc.GetClientID()),
				slog.String("Event", sseMsg.EventType),
				slog.Int("N bytes", n))
		}
		w.(http.Flusher).Flush()
	}
	//cs.DeleteSSEChan(sseChan)
	slog.Debug("SseConnection.Serve: sse connection closed",
		slog.String("remote", r.RemoteAddr),
		slog.String("clientID", sc.GetClientID()),
		slog.String("connectionID", sc.ConnectionID),
	)
}

// SetTimeout set the timeout sending requests
// func (sc *SseScServerConnection) SetTimeout(timeout time.Duration) {
// 	sc.respTimeout = timeout
// }

// NewSseScServerConnection creates a new SSE 1-way connection instance.
// This implements the IConnection interface.
//
// clientID is the authenticated ID of the client that just connected
// connectionID is the client's instance connectionID
// remoteAdd is the address used to connect.
// httpReq is the request that started the websocket connection
// rnrChan is the http server request&response channel where responses are passed.
func NewSseScServerConnection(
	clientID string, connectionID string, remoteAddr string,
	httpReq *http.Request,
	reqHandler msg.RequestHandler,
	notifHandler msg.NotificationHandler,
) *SseScServerConnection {

	c := &SseScServerConnection{
		remoteAddr:   remoteAddr,
		httpReq:      httpReq,
		lastActivity: time.Now(),
	}
	encoder := transport.NewRRNJsonEncoder()

	// OnRemoteMessage is not used so requestHandler and notificationHandler can be nil.
	c.ServerConnectionBase = transport.NewServerConnectionBase(
		clientID, remoteAddr, connectionID,
		encoder, c._sendRaw, reqHandler, notifHandler,
	)

	// interface check
	var _ api.IConnection = c
	return c
}
