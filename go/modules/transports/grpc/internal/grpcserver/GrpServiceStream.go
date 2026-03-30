package grpcserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"time"

	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
)

const ServiceMsgChanSize = 30

// GrpcServiceStream for receiving and sending messages on a GRPC stream connection.
// Use Run to start processing incoming messages.
type GrpcServiceStream struct {
	// this channel is used to signal the read/write loops to exit
	cancelChan chan struct{}

	// clientID     string
	// connectionID string
	isConnected atomic.Bool
	grpcStream  grpcapi.GrpcService_MsgStreamServer

	// handler to pass received messages to
	// recvHandler func(msgType string, jsonRaw string)
	// notifHandler handles the requests received from the remote producer
	// notifHandler msg.NotificationHandler

	// reqHandler handles the requests received from the remote consumer
	// reqHandler msg.RequestHandler

	// request-response channel used to server request replyTo callbacks
	// rnrChan *msg.RnRChan

	// how long to wait for a response after sending a request
	respTimeout time.Duration

	// the send channel with buffer to force sequential sending
	sendChan chan *grpcapi.GrpcMsg
}

// when the client disconnects, we want to make sure that the read loop exits gracefully and that all pending response handlers are notified of the disconnection. This is done by cancelling the stream context, which should cause the read loop to exit with a context.Canceled error. The response handlers will then be notified of the disconnection and can handle it accordingly.
func (sc *GrpcServiceStream) cancelSafely() {
	if !sc.isConnected.Load() {
		return
	}
	sc.isConnected.Store(false)
	close(sc.cancelChan)
	close(sc.sendChan)
}

// Start Processing a message stream from the client.
// This returns when the stream is closed.
func (sc *GrpcServiceStream) recvLoop(recvHandler func(msgType string, jsonRaw string)) {
	// see also https://stackoverflow.com/questions/46933538/how-to-close-grpc-stream-for-server
	for {
		result, err := sc.grpcStream.Recv()
		if err == io.EOF {
			slog.Info("service recvLoop: grpc stream read loop closed due to EOF")
			break
		} else if errors.Is(err, context.Canceled) {
			slog.Info("service recvLoop: Graceful shutdown")
			break
		} else if err != nil {
			slog.Warn("service recvLoop: Recv error", "err", err.Error())
			break
		}
		slog.Info("service recvLoop: received message:" + result.MsgType)
		// parent handles flow control
		recvHandler(result.MsgType, result.JsonPayload)
	}
	slog.Info("service recvLoop: recvLoop ended")
	sc.cancelSafely() // in case client disconnected
}

// Send a stream message to the remote client
func (sc *GrpcServiceStream) Send(msgType, jsonPayload string) (err error) {
	// if !sc.isConnected.Load() {
	// 	return grpcapi.ErrConnectionClosed
	// }
	// FIXME: prevent a race between closing the send channel and writing to it
	grpcMsg := &grpcapi.GrpcMsg{
		MsgType:     msgType,
		JsonPayload: jsonPayload,
	}
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
	select {
	case sc.sendChan <- grpcMsg:
		// all is well
		err = nil
	case <-ctx.Done():
		err = ctx.Err()
	default:
		// the client is too slow -- disconnect it
		sc.cancelSafely()
		err = grpcapi.ErrClientTooSlow
	}
	cancelFn()
	return err
}

// Send loop using the send channel to ensure sequential delivery of messages.
// grpc streams do not support concurrent sending.
func (sc *GrpcServiceStream) sendLoop() {
	for msg := range sc.sendChan {
		slog.Info("service sendLoop: sending message:" + msg.MsgType)
		if err := sc.grpcStream.Send(msg); err != nil {
			break
		}
	}
	slog.Info("service sendLoop: sendLoop ended")
	sc.cancelSafely() // in	case client disconnected
}

// Close the stream connection
func (sc *GrpcServiceStream) Close() {
	sc.cancelSafely()
}

// // IsConnected returns the current connection status
// func (sc *GrpcServerConnection) IsConnected() bool {
// 	return sc.isConnected.Load()
// }

// Run starts processing a message stream from the client.
// the recvHandler receives messages as they come in and is responsible for flow
// control by not blocking the read loop.
// The send loop ensures that messages are sent sequentially, as required by grpc streams.
// The Run function will return when the stream is closed, either by the client or by the server.
func (sc *GrpcServiceStream) Run(recvHandler func(msgType string, jsonRaw string)) {
	// default buffer size. TODO:flow control
	sc.sendChan = make(chan *grpcapi.GrpcMsg, ServiceMsgChanSize)
	sc.cancelChan = make(chan struct{})
	sc.isConnected.Store(true)
	go sc.recvLoop(recvHandler)
	go sc.sendLoop()
	<-(sc.cancelChan)
	sc.isConnected.Store(false)
}

// Create a server side client connection of a grpc messaging stream
//
// Run Run() to start processing the stream.
func NewGrpcServiceStream(
	grpcStream grpcapi.GrpcService_MsgStreamServer,
	respTimeout time.Duration,
) *GrpcServiceStream {
	strm := &GrpcServiceStream{
		// clientID:     clientID,
		// connectionID: connectionID,
		// reqHandler:   reqHandler,
		// notifHandler: notifHandler,
		respTimeout: respTimeout,
		grpcStream:  grpcStream,
	}
	// peerInfo, ok := peer.FromContext(grpcStream.Context())
	// var remoteAddr string
	// if ok {
	// 	remoteAddr = peerInfo.Addr.String()
	// }
	return strm
}
