package grpcserver

import (
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
	"github.com/hiveot/hivekit/go/modules/transports/grpc/internal/grpcclient"
	"github.com/hiveot/hivekit/go/msg"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"google.golang.org/grpc/peer"
)

// TransportConnection implements the IConnection interface
// Intended to provide RRN messages for an incoming GRPC stream connection.
// Use ReadLoop to start processing incoming messages.
type TransportConnection struct {
	transports.ServerConnectionBase

	bstrm *grpcclient.BufferedStream

	// notifHandler handles the requests received from the remote producer
	notifHandler msg.NotificationHandler

	// reqHandler handles the requests received from the remote consumer
	reqHandler msg.RequestHandler

	// request-response channel used to server request replyTo callbacks
	rnrChan *msg.RnRChan

	// how long to wait for a response after sending a request
	respTimeout time.Duration
}

// when the client disconnects, we want to make sure that the read loop exits gracefully and that all pending response handlers are notified of the disconnection. This is done by cancelling the stream context, which should cause the read loop to exit with a context.Canceled error. The response handlers will then be notified of the disconnection and can handle it accordingly.
// func (sc *TransportConnection) cancelSafely() {
// 	sc.isConnected.Store(false)
// 	close(sc.cancelChan)
// }

// onMessage handles an incoming message
// The message is converted into a request, response or notification and passed
// on to the registered handler.
func (sc *TransportConnection) onMessage(msgType string, jsonRaw string) {

	switch msgType {
	case msg.MessageTypeNotification:
		var notifMsg msg.NotificationMessage
		err := jsoniter.UnmarshalFromString(jsonRaw, &notifMsg)
		if err != nil {
			slog.Error("Failed to unmarshal notification message", "err", err.Error())
			return
		}
		sc.notifHandler(&notifMsg)

	case msg.MessageTypeRequest:
		var reqMsg msg.RequestMessage
		err := jsoniter.UnmarshalFromString(jsonRaw, &reqMsg)
		if err != nil {
			slog.Error("Failed to unmarshal request message", "err", err.Error())
			return
		}
		sc.reqHandler(&reqMsg, func(reply *msg.ResponseMessage) error {
			if reply != nil {
				err = sc.SendResponse(reply)
			} else {
				slog.Error("_onMessage (request): reply callback without response")
			}
			return err
		})
	case msg.MessageTypeResponse:
		var resp msg.ResponseMessage
		err := jsoniter.UnmarshalFromString(jsonRaw, &resp)
		if err != nil {
			slog.Error("Failed to unmarshal response message", "err", err.Error())
			return
		}
		handled := sc.rnrChan.HandleResponse(&resp, sc.respTimeout)
		if !handled {
			slog.Warn("_onMessage: No response handler for request, response is lost",
				"correlationID", resp.CorrelationID,
				"op", resp.Operation,
				"thingID", resp.ThingID,
				"name", resp.Name)
		}
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

func (sc *TransportConnection) SendNotification(notif *msg.NotificationMessage) {
	JsonPayload, _ := jsoniter.MarshalToString(notif)
	err := sc.bstrm.Send(msg.MessageTypeNotification, JsonPayload)
	if err != nil {
		slog.Error("Failed to send notification message", "err", err.Error())
	}
}
func (sc *TransportConnection) SendRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	JsonPayload, _ := jsoniter.MarshalToString(req)

	// without a replyTo simply send the request
	if replyTo == nil {
		err := sc.bstrm.Send(msg.MessageTypeRequest, JsonPayload)
		return err
	}

	// with a replyTo, send the response async to the replyTo handler
	// catch the response in a channel linked by correlation-id
	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	// this is non-blocking. A reply will be passed to the replyTo callback
	sc.rnrChan.WaitWithCallback(req.CorrelationID, sc.respTimeout, replyTo)
	// now the RNR channel is ready, send the request message
	err := sc.bstrm.Send(msg.MessageTypeRequest, JsonPayload)
	return err
}

func (sc *TransportConnection) SendResponse(response *msg.ResponseMessage) error {
	// TODO: put grpc stuff in a message converter and standardize more of
	// the connection logic in ServerConnectionBase.
	// msg, err := c.messageConverter.EncodeResponse(response)
	JsonPayload, _ := jsoniter.MarshalToString(response)
	err := sc.bstrm.Send(msg.MessageTypeResponse, JsonPayload)
	if err != nil {
		slog.Error("Failed to send response message", "err", err.Error())
	}
	return err
}
func (sc *TransportConnection) SetTimeout(timeout time.Duration) {
	sc.respTimeout = timeout
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
	respTimeout time.Duration,
) *TransportConnection {

	c := &TransportConnection{
		reqHandler:   reqHandler,
		notifHandler: notifHandler,
		respTimeout:  respTimeout,
		rnrChan:      msg.NewRnRChan(),
	}
	// // use the same buffered stream as the client uses for sending and receiving messages
	c.bstrm = grpcclient.NewGrpcBufferedStream(grpcStream, c.onMessage, time.Minute)

	// determine the client ID and connection ID from the grpc stream context
	peerInfo, ok := peer.FromContext(grpcStream.Context())
	var remoteAddr string
	if ok {
		remoteAddr = peerInfo.Addr.String()
	}
	c.Init(clientID, remoteAddr, connectionID)
	var _ transports.IConnection = c // interface check

	return c
}
