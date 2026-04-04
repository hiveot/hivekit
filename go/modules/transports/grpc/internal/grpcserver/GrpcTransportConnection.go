package grpcserver

import (
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
	"github.com/hiveot/hivekit/go/modules/transports/grpc/internal"
	"github.com/hiveot/hivekit/go/msg"
	"google.golang.org/grpc/peer"
)

// TransportConnection implements the IConnection interface
// Intended to provide RRN messages for an incoming GRPC stream connection.
// Use ReadLoop to start processing incoming messages.
type TransportConnection struct {
	transports.ServerConnectionBase

	bstrm *internal.BufferedStream

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
func (sc *TransportConnection) _onMessage(msgType string, raw []byte) {

	switch msgType {
	case msg.MessageTypeNotification:
		notif, err := sc.encoder.DecodeNotification(raw)
		if err != nil {
			slog.Error("Failed to unmarshal notification message", "err", err.Error())
			return
		}
		notif.SenderID = sc.GetClientID()
		sc.OnNotification(notif, sc.notifHandler)

	case msg.MessageTypeRequest:
		req, err := sc.encoder.DecodeRequest(raw)
		if err != nil {
			slog.Error("Failed to unmarshal request message", "err", err.Error())
			return
		}
		req.SenderID = sc.GetClientID()
		sc.OnRequest(req, sc.reqHandler)

	case msg.MessageTypeResponse:
		resp, err := sc.encoder.DecodeResponse(raw)
		if err != nil {
			slog.Error("Failed to unmarshal response message", "err", err.Error())
			return
		}
		resp.SenderID = sc.GetClientID()
		sc.OnResponse(resp)

	}
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
	return sc.bstrm.Send(msgType, raw)
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
	grpcStream grpcapi.GrpcService_MsgStreamServer,
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
	// // use the same buffered stream as the client uses for sending and receiving messages
	c.bstrm = internal.NewGrpcBufferedStream(grpcStream, c._onMessage, time.Minute)

	// determine the client ID and connection ID from the grpc stream context
	peerInfo, ok := peer.FromContext(grpcStream.Context())
	var remoteAddr string
	if ok {
		remoteAddr = peerInfo.Addr.String()
	}
	c.Init(clientID, remoteAddr, connectionID, nil, c.sendRaw)
	var _ transports.IConnection = c // interface check

	return c
}
