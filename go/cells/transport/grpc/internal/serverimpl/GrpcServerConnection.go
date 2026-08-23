package serverimpl

import (
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells/transport"
	grpclib "github.com/hiveot/hivekit/go/cells/transport/grpc/internal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

// GrpcServerConnection implements the IConnection interface for grpc server side connections.
// Intended to handle RRN messages for a GRPC stream connection.
// Use ReadLoop to start processing incoming messages.
type GrpcServerConnection struct {
	// ServerConnectionBase handles the generic messaging part
	*transport.ServerConnectionBase

	// buffered stream handles the raw grpc data
	bstrm *grpclib.BufferedStream
}

func (sc *GrpcServerConnection) _sendRaw(msgType string, raw []byte) error {
	return sc.bstrm.Send(raw)
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
func (sc *GrpcServerConnection) GetConnectionStatus() api.ConnectionStatus {
	if sc.bstrm.IsConnected() {
		return api.StatusConnected
	}
	return api.StatusLost
}

// Run starts processing a message stream from the client.
// This returns when the stream is closed.
func (sc *GrpcServerConnection) WaitUntilDisconnect() {
	sc.bstrm.WaitUntilDisconnect()
}

// Create a transport server side connection of a grpc messaging stream.
// This implemements the IConnection interface.
//
// Use Close() to close the connection from the server end.
// Run WaitUntilDisconnect() to block until the connection is closed by the client or server.
func NewGrpcServerConnection(
	clientID string,
	connectionID string,
	grpcStream grpc.ServerStream,
	reqHandler msg.RequestHandler,
	notifHandler msg.NotificationHandler,
	// respTimeout time.Duration,
) *GrpcServerConnection {

	slog.Info("StartGrpcTransportConnection", slog.String("clientID", clientID))
	// determine the client ID and connection ID from the grpc stream context
	peerInfo, ok := peer.FromContext(grpcStream.Context())
	var remoteAddr string
	if ok {
		remoteAddr = peerInfo.Addr.String()
	}

	c := &GrpcServerConnection{}
	// the server connection base handles the generic portion of message handling
	encoder := transport.NewRRNJsonEncoder()
	c.ServerConnectionBase = transport.NewServerConnectionBase(
		clientID, remoteAddr, connectionID,
		encoder, c._sendRaw, reqHandler, notifHandler,
	)

	// use the same buffered stream as the client uses for sending and receiving messages
	c.bstrm = grpclib.NewBufferedStream(grpcStream, nil, c.OnRemoteMessage, time.Minute)

	var _ api.IConnection = c // interface check

	return c
}
