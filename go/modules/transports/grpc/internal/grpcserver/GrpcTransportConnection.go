package grpcserver

import (
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
	"github.com/hiveot/hivekit/go/modules/transports/grpc/internal"
	"github.com/hiveot/hivekit/go/msg"
	jsoniter "github.com/json-iterator/go"
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

	// request-response channel used to server request replyTo callbacks
	// rnrChan *msg.RnRChan

	// how long to wait for a response after sending a request
	// respTimeout time.Duration
}

// when the client disconnects, we want to make sure that the read loop exits gracefully and that all pending response handlers are notified of the disconnection. This is done by cancelling the stream context, which should cause the read loop to exit with a context.Canceled error. The response handlers will then be notified of the disconnection and can handle it accordingly.
// func (sc *TransportConnection) cancelSafely() {
// 	sc.isConnected.Store(false)
// 	close(sc.cancelChan)
// }

// _onMessage handles an incoming message
// The message is converted into a request, response or notification and passed
// on to the registered handler.
func (sc *TransportConnection) _onMessage(msgType string, jsonRaw string) {

	switch msgType {
	case msg.MessageTypeNotification:
		var notifMsg msg.NotificationMessage
		err := jsoniter.UnmarshalFromString(jsonRaw, &notifMsg)
		if err != nil {
			slog.Error("Failed to unmarshal notification message", "err", err.Error())
			return
		}
		sc.OnNotification(&notifMsg, sc.notifHandler)

	case msg.MessageTypeRequest:
		var req msg.RequestMessage
		err := jsoniter.UnmarshalFromString(jsonRaw, &req)
		if err != nil {
			slog.Error("Failed to unmarshal request message", "err", err.Error())
			return
		}
		req.SenderID = sc.GetClientID()
		sc.OnRequest(&req, sc.reqHandler)

	case msg.MessageTypeResponse:
		var resp msg.ResponseMessage
		err := jsoniter.UnmarshalFromString(jsonRaw, &resp)
		if err != nil {
			slog.Error("Failed to unmarshal response message", "err", err.Error())
			return
		}
		sc.OnResponse(&resp)
		// handled := sc.rnrChan.HandleResponse(&resp, sc.respTimeout)
		// if !handled {
		// 	slog.Warn("_onMessage: No response handler for request, response is lost",
		// 		"correlationID", resp.CorrelationID,
		// 		"op", resp.Operation,
		// 		"thingID", resp.ThingID,
		// 		"name", resp.Name)
		// }
	}
}

// Close the stream connection
func (sc *TransportConnection) Close() {
	sc.bstrm.Close()
}

// func (sc *TransportConnection) GetClientID() string {
// 	return sc.clientID
// }
// func (sc *TransportConnection) GetConnectionID() string {
// 	return sc.connectionID
// }

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
