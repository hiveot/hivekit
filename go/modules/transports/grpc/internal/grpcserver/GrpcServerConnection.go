package grpcserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
	"github.com/hiveot/hivekit/go/msg"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"google.golang.org/grpc/peer"
)

// GrpcServerConnection implements the IConnection interface
// Intended to provide RRN messages for an incoming GRPC stream connection.
// Use ReadLoop to start processing incoming messages.
type GrpcServerConnection struct {
	transports.ServerConnectionBase
	// this channel is used to signal the read/write loops to exit
	cancelChan chan struct{}

	clientID     string
	connectionID string
	isConnected  atomic.Bool
	grpcStream   grpcapi.GrpcService_MsgStreamServer

	// notifHandler handles the requests received from the remote producer
	notifHandler msg.NotificationHandler

	// reqHandler handles the requests received from the remote consumer
	reqHandler msg.RequestHandler

	// request-response channel used to server request replyTo callbacks
	rnrChan *msg.RnRChan

	// how long to wait for a response after sending a request
	respTimeout time.Duration

	// the send channel with buffer to force sequential sending
	sendChan chan *grpcapi.GrpcMsg
}

// when the client disconnects, we want to make sure that the read loop exits gracefully and that all pending response handlers are notified of the disconnection. This is done by cancelling the stream context, which should cause the read loop to exit with a context.Canceled error. The response handlers will then be notified of the disconnection and can handle it accordingly.
func (sc *GrpcServerConnection) cancelSafely() {
	sc.isConnected.Store(false)
	close(sc.cancelChan)
}

// onMessage handles an incoming message
// The message is converted into a request, response or notification and passed
// on to the registered handler.
func (sc *GrpcServerConnection) onMessage(msgType string, jsonRaw string) {

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

// Start Processing a message stream from the client.
// This returns when the stream is closed.
func (sc *GrpcServerConnection) readLoop() {
	// see also https://stackoverflow.com/questions/46933538/how-to-close-grpc-stream-for-server
	for {
		result, err := sc.grpcStream.Recv()
		if err == io.EOF {
			slog.Info("Server RecvLoop: grpc stream read loop closed due to EOF")
			break
		} else if errors.Is(err, context.Canceled) {
			slog.Info("Server RecvLoop: Graceful shutdown")
			break
		} else if err != nil {
			slog.Warn("Server RecvLoop: Recv error", "err", err.Error())
			break
		}
		slog.Info("received message:" + result.MsgType)
		// parent handles flow control
		sc.onMessage(result.MsgType, result.JsonPayload)
	}
	sc.cancelSafely() // in case client disconnected
}

// Send a stream message to the remote client
func (sc *GrpcServerConnection) send(msgType, jsonPayload string) (err error) {
	if !sc.isConnected.Load() {
		return grpcapi.ErrConnectionClosed
	}
	// FIXME: prevent a race between closing the send channel and writing to it
	grpcMsg := &grpcapi.GrpcMsg{
		MsgType:     msgType,
		JsonPayload: jsonPayload,
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFn()
	select {
	case sc.sendChan <- grpcMsg:
		// all is well
	case <-ctx.Done():
		return ctx.Err()
	default:
		// the client is too slow -- disconnect it
		sc.cancelSafely()
		return grpcapi.ErrClientTooSlow
	}
	return nil
}

// Send loop using the send channel to ensure sequential delivery of messages.
// grpc streams do not support concurrent sending.
func (sc *GrpcServerConnection) sendLoop() {
	for msg := range sc.sendChan {
		if err := sc.grpcStream.Send(msg); err != nil {
			break
		}
	}
	sc.cancelSafely() // in	case client disconnected
}

// Close the stream connection
func (sc *GrpcServerConnection) Close() {
	sc.cancelSafely()
}

func (sc *GrpcServerConnection) GetClientID() string {
	return sc.clientID
}
func (sc *GrpcServerConnection) GetConnectionID() string {
	return sc.connectionID
}

// IsConnected returns the current connection status
func (sc *GrpcServerConnection) IsConnected() bool {
	return sc.isConnected.Load()
}

// Run starts processing a message stream from the client.
// This returns when the stream is closed.
func (sc *GrpcServerConnection) Run() {
	sc.cancelChan = make(chan struct{})
	sc.isConnected.Store(true)
	go sc.readLoop()
	go sc.sendLoop()
	<-sc.cancelChan
}

func (sc *GrpcServerConnection) SendNotification(notif *msg.NotificationMessage) {
	JsonPayload, _ := jsoniter.MarshalToString(notif)
	err := sc.send(msg.MessageTypeNotification, JsonPayload)
	if err != nil {
		slog.Error("Failed to send notification message", "err", err.Error())
	}
}
func (sc *GrpcServerConnection) SendRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	JsonPayload, _ := jsoniter.MarshalToString(req)

	// without a replyTo simply send the request
	if replyTo == nil {
		err := sc.send(msg.MessageTypeRequest, JsonPayload)
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
	err := sc.send(msg.MessageTypeRequest, JsonPayload)
	return err
}

func (sc *GrpcServerConnection) SendResponse(response *msg.ResponseMessage) error {
	// TODO: put grpc stuff in a message converter and standardize more of
	// the connection logic in ServerConnectionBase.
	// msg, err := c.messageConverter.EncodeResponse(response)
	JsonPayload, _ := jsoniter.MarshalToString(response)
	err := sc.send(msg.MessageTypeResponse, JsonPayload)
	if err != nil {
		slog.Error("Failed to send response message", "err", err.Error())
	}
	return err
}
func (sc *GrpcServerConnection) SetTimeout(timeout time.Duration) {
	sc.respTimeout = timeout
}

// Create a server side client connection of a grpc messaging stream
// This implemements the IConnection interface.
//
// Run Run() to start processing the stream.
func NewGrpcServerConnection(
	clientID string,
	connectionID string,
	reqHandler msg.RequestHandler,
	notifHandler msg.NotificationHandler,
	respTimeout time.Duration,
	grpcStream grpcapi.GrpcService_MsgStreamServer,
) *GrpcServerConnection {
	c := &GrpcServerConnection{
		clientID:     clientID,
		connectionID: connectionID,
		reqHandler:   reqHandler,
		notifHandler: notifHandler,
		respTimeout:  respTimeout,
		grpcStream:   grpcStream,
		// default buffer size. TODO:flow control
		sendChan: make(chan *grpcapi.GrpcMsg, 30),
	}
	peerInfo, ok := peer.FromContext(grpcStream.Context())
	var remoteAddr string
	if ok {
		remoteAddr = peerInfo.Addr.String()
	}
	c.Init(clientID, remoteAddr, connectionID)
	var _ transports.IConnection = c // interface check

	return c
}
