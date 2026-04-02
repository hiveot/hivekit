package grpcclient

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"time"

	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
)

// The size of the send channel buffer
const SendChanSize = 30

// BufferedStream for receiving and sending messages on a GRPC stream connection.
// Use Run to start processing incoming messages.
//
// The send loop uses an adaptive delay that increases when the send buffer fills up to 50% level
// and lowers when the send buffer is below 30% the delay decreases but at 10% if the increase.
type BufferedStream struct {

	// this channel is used to signal the read/write loops to exit
	cancelChan chan struct{}

	isConnected atomic.Bool
	grpcStream  grpcapi.IGrpcMessageStream

	// how long to wait for a response after sending a request
	sendTimeout time.Duration

	// the send channel with buffer to force sequential sending
	sendChan chan *grpcapi.GrpcMsg

	// backoff flow control timer to delay sending
	// This increases by 1us when the buffer is full and decreases after a successful sent
	fcDelay time.Duration
}

// Start Processing a message stream from the client.
// This returns when the stream is closed.
func (bs *BufferedStream) recvLoop(recvHandler func(msgType string, jsonRaw string)) {
	// see also https://stackoverflow.com/questions/46933538/how-to-close-grpc-stream-for-server
	for {
		if !bs.isConnected.Load() {
			break
		}
		result, err := bs.grpcStream.Recv()
		if err == io.EOF {
			slog.Debug("service recvLoop: grpc stream read loop closed due to EOF")
			break
		} else if errors.Is(err, context.Canceled) {
			slog.Debug("service recvLoop: Graceful shutdown")
			break
		} else if err != nil {
			slog.Info("service recvLoop: Recv error", "err", err.Error())
			break
		}
		slog.Debug("service recvLoop: received message:" + result.MsgType)
		// parent handles flow control
		recvHandler(result.MsgType, result.JsonPayload)
	}
	slog.Debug("service recvLoop: recvLoop ended")

	//end the blocking in WaitUntilDisconnect
	bs.Close()
}

// Send loop using the send channel to ensure sequential delivery of messages.
// grpc streams do not support concurrent sending.
func (bs *BufferedStream) sendLoop() {
	for msg := range bs.sendChan {
		slog.Debug("service sendLoop: sending message:" + msg.MsgType)
		if err := bs.grpcStream.Send(msg); err != nil {
			break
		}
	}
	slog.Debug("service sendLoop: sendLoop ended")
	// bs.cancelSafely() // in	case client disconnected
}

// Close the stream connection
func (bs *BufferedStream) Close() {
	if !bs.isConnected.Load() {
		return
	}
	bs.isConnected.Store(false)
	close(bs.cancelChan)
}

// // IsConnected returns the current connection status
func (sc *BufferedStream) IsConnected() bool {
	return sc.isConnected.Load()
}

// Send a stream message to the remote client.
// This uses an adaptive delay when the send buffer passes 50%.
//
// If the buffer fills up then the send is retried 10 times, each with a (10usec) larger delay.
// If this still fails then the client connection is broken or the receiver is stuck. In that
// case this returns an error and the sender should consider closing the connection.
func (bs *BufferedStream) Send(msgType, jsonPayload string) (err error) {
	const MaxRetryCount = 10
	grpcMsg := &grpcapi.GrpcMsg{
		MsgType:     msgType,
		JsonPayload: jsonPayload,
	}
	// if the send channel is full, allow a wait before disconnecting the client.
	ctx, cancelFn := context.WithTimeout(context.Background(), bs.sendTimeout)
	defer cancelFn()

	// if the buffer is full then the remote client is considered stuck. The caller should close
	// the connection.
	for retryCount := MaxRetryCount; retryCount > 0; retryCount-- {
		// slog.Info("- Send", "retrycnt", retryCount, "delay", bs.fcDelay, "chan level", len(bs.sendChan))

		// These self-balancing numbers are intended for UDS which is faster than tcp.
		// if the send buffer fills up to 50% then increase the delay with a microsecond.
		// if the send buffer falls below 30% then decrease the delay with 0.1 microsecond.
		if len(bs.sendChan) > SendChanSize/2 {
			bs.fcDelay += time.Microsecond
		} else if bs.fcDelay > 0 && len(bs.sendChan) < SendChanSize/3 {
			bs.fcDelay = bs.fcDelay - time.Nanosecond*100
		}
		if bs.fcDelay > 0 {
			time.Sleep(bs.fcDelay)
		}
		select {
		case bs.sendChan <- grpcMsg:
			// all is well
			err = nil
			retryCount = 0 // end the retry loop
		case <-ctx.Done():
			err = ctx.Err()
			retryCount = 0
		default:
			// if the channel is full then retry, but add a substantial delay
			bs.fcDelay = bs.fcDelay + time.Microsecond*10
			slog.Warn("Send: channel is full. increasing sent delay to ", "delay", bs.fcDelay)
			err = grpcapi.ErrClientTooSlow
			if retryCount == 1 {
				// ideally this never happens
				slog.Error("Failed to send. Buffer is full.",
					"retryCount", retryCount, "delay", bs.fcDelay)
			}
		}
		if err == nil {
			break
		}
	}
	cancelFn()
	return err
}

// WaitUntilDisconnect waits until the send or receive stream is closed.
// Intended to be called by the (server) serve handler to avoid the stream from
// closing on return.
func (bs *BufferedStream) WaitUntilDisconnect() {
	<-(bs.cancelChan)
	bs.isConnected.Store(false)
}

// NewGrpcBufferedStream creates a concurrently safe buffered instance from a grpc stream.
//
// The resulting instance supports concurrent sending and receiving of messages.
// This supports sending messages concurrently.
// If the send buffer is full, the client is considered too slow and the stream is closed
// after waiting the sendTimeout duration for the buffer to be available.
//
// Call WaitUntilDisconnect() after creating this instance to wait until the stream
// is closed by the client or the server. Needed by the server serve handler.
//
//	grpcStream is the raw GRPC stream (currently generated by protobuf)
//	recvHandler is called when a new message is received from the stream.
//	sendTimeout is the default timeout for sending messages on this stream
//	 when the send buffer is full.
func NewGrpcBufferedStream(
	grpcStream grpcapi.IGrpcMessageStream,
	recvHandler func(msgType string, jsonRaw string),
	sendTimeout time.Duration,
) *BufferedStream {
	strm := &BufferedStream{
		sendTimeout: sendTimeout,
		grpcStream:  grpcStream,
		// ServiceMsgChanSize is the default buffer size.
		sendChan:   make(chan *grpcapi.GrpcMsg, SendChanSize),
		cancelChan: make(chan struct{}),
	}
	strm.isConnected.Store(true)
	go strm.recvLoop(recvHandler)
	go strm.sendLoop()

	// peerInfo, ok := peer.FromContext(grpcStream.Context())
	// var remoteAddr string
	// if ok {
	// 	remoteAddr = peerInfo.Addr.String()
	// }
	return strm
}
