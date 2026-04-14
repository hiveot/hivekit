package grpcserver

import (
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules/transports"
	grpclib "github.com/hiveot/hivekit/go/modules/transports/grpc/lib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

// TransportConnection implements the IConnection interface
// Intended to provide RRN messages for an incoming GRPC stream connection.
// Use ReadLoop to start processing incoming messages.
type TransportConnection struct {
	transports.ServerConnectionBase

	bstrm *grpclib.BufferedStream

	// notifHandler handles the requests received from the remote producer
	notifHandler msg.NotificationHandler

	// reqHandler handles the requests received from the remote consumer
	reqHandler msg.RequestHandler

	encoder transports.IMessageEncoder
	// request-response channel used to server request replyTo callbacks
	// rnrChan *msg.RnRChan

	// how long to wait for a response after sending a request
	// respTimeout time.Duration
}

// _onMessage handles an incoming message
// The message is converted into a request, response or notification and passed
// on to the registered handler.
func (sc *TransportConnection) _onServerMessage(raw []byte) {

	if raw == nil {
		slog.Error("_onServerMessage: raw data is nil")
	}
	notif, err := sc.encoder.DecodeNotification(raw)
	if err == nil {
		notif.SenderID = sc.GetClientID()
		sc.OnNotification(notif, sc.notifHandler)
		return
	}
	req, err := sc.encoder.DecodeRequest(raw)
	if err == nil {
		req.SenderID = sc.GetClientID()
		sc.OnRequest(req, sc.reqHandler)
		return
	}
	resp, err := sc.encoder.DecodeResponse(raw)
	if err == nil {
		resp.SenderID = sc.GetClientID()
		sc.OnResponse(resp)
		return
	}
	slog.Error("_onServerMessage: Failed to unmarshal message", "err", err.Error())
}

// Close the stream connection
func (sc *TransportConnection) Close() {
	sc.bstrm.Close()
}

// IsConnected returns the current connection status
func (sc *TransportConnection) IsConnected() bool {
	return sc.bstrm.IsConnected()
}

func (sc *TransportConnection) sendRaw(msgType string, raw []byte) error {
	return sc.bstrm.Send(raw)
}

// Run starts processing a message stream from the client.
// This returns when the stream is closed.
func (sc *TransportConnection) WaitUntilDisconnect() {
	sc.bstrm.WaitUntilDisconnect()
}

// Create a transport server side connection of a grpc messaging stream.
// This implemements the IConnection interface.
//
// Run Close() to close the connection from the server end
// Run WaitUntilDisconnect() to block until the connection is closed by the client or server.
func StartGrpcTransportConnection(
	clientID string,
	connectionID string,
	grpcStream grpc.ServerStream,
	reqHandler msg.RequestHandler,
	notifHandler msg.NotificationHandler,
	// respTimeout time.Duration,
) *TransportConnection {

	c := &TransportConnection{
		reqHandler:   reqHandler,
		notifHandler: notifHandler,
		encoder:      transports.NewRRNJsonEncoder(),
		// respTimeout:  respTimeout,
		// rnrChan:      msg.NewRnRChan(),
	}
	// determine the client ID and connection ID from the grpc stream context
	peerInfo, ok := peer.FromContext(grpcStream.Context())
	var remoteAddr string
	if ok {
		remoteAddr = peerInfo.Addr.String()
	}
	c.Init(clientID, remoteAddr, connectionID, nil, c.sendRaw)

	// // use the same buffered stream as the client uses for sending and receiving messages
	c.bstrm = grpclib.NewBufferedStream(grpcStream, nil, c._onServerMessage, time.Minute)

	var _ transports.IConnection = c // interface check

	return c
}
