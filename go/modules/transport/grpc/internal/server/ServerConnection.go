package internalserver

import (
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules/transport"
	grpclib "github.com/hiveot/hivekit/go/modules/transport/grpc/internal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

// GrpcServerConnection implements the IConnection interface
// Intended to provide RRN messages for an incoming GRPC stream connection.
// Use ReadLoop to start processing incoming messages.
type GrpcServerConnection struct {
	transport.ServerConnectionBase

	bstrm *grpclib.BufferedStream

	// notifHandler handles the requests received from the remote producer
	notifHandler msg.NotificationHandler

	// reqHandler handles the requests received from the remote consumer
	reqHandler msg.RequestHandler

	encoder transport.IMessageEncoder
}

// _onMessage handles an incoming message
// The message is converted into a request, response or notification and passed
// on to the registered handler.
func (sc *GrpcServerConnection) _onServerMessage(raw []byte) {

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
func (sc *GrpcServerConnection) Close() {
	sc.bstrm.Close()
}

// IsConnected returns the current connection status
func (sc *GrpcServerConnection) IsConnected() bool {
	return sc.bstrm.IsConnected()
}

// // GetConnectionStatus returns the current connection status
func (sc *GrpcServerConnection) GetConnectionStatus() transport.ConnectionStatus {
	if sc.bstrm.IsConnected() {
		return transport.StatusConnected
	}
	return transport.StatusLost
}

func (sc *GrpcServerConnection) sendRaw(msgType string, raw []byte) error {
	return sc.bstrm.Send(raw)
}

// Run starts processing a message stream from the client.
// This returns when the stream is closed.
func (sc *GrpcServerConnection) WaitUntilDisconnect() {
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
) *GrpcServerConnection {

	slog.Info("StartGrpcTransportConnection", slog.String("clientID", clientID))
	c := &GrpcServerConnection{
		reqHandler:   reqHandler,
		notifHandler: notifHandler,
		encoder:      transport.NewRRNJsonEncoder(),
	}
	// determine the client ID and connection ID from the grpc stream context
	peerInfo, ok := peer.FromContext(grpcStream.Context())
	var remoteAddr string
	if ok {
		remoteAddr = peerInfo.Addr.String()
	}
	c.Init(clientID, remoteAddr, connectionID, nil, c.sendRaw)

	// use the same buffered stream as the client uses for sending and receiving messages
	c.bstrm = grpclib.NewBufferedStream(grpcStream, nil, c._onServerMessage, time.Minute)

	var _ transport.IConnection = c // interface check

	return c
}
