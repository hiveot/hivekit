package grpclib

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// error result codes
var ErrConnectionClosed = status.Errorf(codes.Canceled, "connection is closed")
var ErrClientTooSlow = status.Errorf(codes.ResourceExhausted, "client is too slow to receive messages")

// The size of the send channel buffer
const SendChanSize = 42

// these timings are arbitrary and depend on the expected receiver performance
const FCDelayIncreaseStep = time.Microsecond * 10
const FCDelayDecreaseStep = time.Microsecond * 3

// API for use by all client/server streaming endpoints
type IMsgStream interface {
	SendMsg(msg any) error
	RecvMsg(dest any) error
}

// BufferedStream for receiving and sending messages on a GRPC stream connection.
// Use Run to start processing incoming messages.
//
// The send loop uses an adaptive delay that increases when the send buffer fills up to 50% level
// and lowers when the send buffer is below 30% the delay decreases but at 10% if the increase.
type BufferedStream struct {

	// flow control delays
	FCDelayDecreaseStep time.Duration
	FCDelayIncreaseStep time.Duration

	// the maximum number of retries once the send buffer is full.
	MaxRetryCount int

	// this channel is used to signal the read/write loops to exit
	cancelChan chan struct{}

	isConnected atomic.Bool

	// the raw GRPC stream
	msgStream IMsgStream

	// cancel function of the raw stream context. Used in client streams.
	msgStreamCancel func()

	// mutex to protect the flow control delay and send channel length checks
	mux sync.Mutex

	// how long to wait for a response after sending a request
	sendTimeout time.Duration

	// the send channel with buffer to force sequential sending
	sendChan chan []byte

	txMsgCnt atomic.Int64

	// backoff flow control timer to delay sending
	// This increases by 1us when the buffer is full and decreases after a successful sent
	fcDelay time.Duration
}

// Start Processing a message stream from the client.
// This returns when the stream is closed.
func (bs *BufferedStream) _recvLoop(recvHandler func(rawMsg []byte)) {
	// see also https://stackoverflow.com/questions/46933538/how-to-close-grpc-stream-for-server
	for {
		var rxMsg []byte
		if !bs.isConnected.Load() {
			break
		}
		err := bs.msgStream.RecvMsg(&rxMsg)
		if err != nil {
			if err == io.EOF {
				slog.Debug("service recvLoop: stream read loop closed due to EOF")
				break
			}
			stat, ok := status.FromError(err)
			if ok && stat.Code() == codes.Canceled {
				slog.Debug("service recvLoop: context cancelled. Graceful shutdown")
				break
			} else {
				slog.Warn("service recvLoop: Recv error", "err", err.Error())
				break
			}
		}
		// received a valid message, pass it to the handler
		recvHandler(rxMsg)
	}
	slog.Debug("service recvLoop: recvLoop ended")

	//end the blocking in WaitUntilDisconnect
	bs.Close()
}

// Send loop using the send channel to ensure sequential delivery of messages.
// grpc streams do not support concurrent sending.
func (bs *BufferedStream) _sendLoop() {
	for msg := range bs.sendChan {
		if err := bs.msgStream.SendMsg(msg); err != nil {
			slog.Error("_sendLoop error", "err", err.Error())
			break
		}
	}
	slog.Debug("service sendLoop: sendLoop ended")
}

// Close the stream and buffered channels
func (bs *BufferedStream) Close() {
	if !bs.isConnected.Load() {
		return
	}
	bs.isConnected.Store(false)
	close(bs.cancelChan)
	// gRPC stream context cancel
	if bs.msgStreamCancel != nil {
		bs.msgStreamCancel()
	}
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
func (bs *BufferedStream) Send(rawMsg []byte) (err error) {

	// if the send channel is full, allow a wait before disconnecting the client.
	ctx, cancelFn := context.WithTimeout(context.Background(), bs.sendTimeout)
	defer cancelFn()

	bs.txMsgCnt.Add(1)

	// if the buffer is full then the remote client is considered stuck. The caller should close
	// the connection.
	for retryCount := 0; ; retryCount++ {
		// These self-balancing numbers are intended for UDS which is faster than tcp.
		// if the send buffer fills up to 50% then increase the delay with a microsecond.
		// if the send buffer falls below 30% then decrease the delay with 0.1 microsecond.
		// if the buffer is empty, reset the delay to 0 to allow burst sending
		bs.mux.Lock()
		fcDelay := bs.fcDelay
		if len(bs.sendChan) == 0 {
			fcDelay = 0
		} else if len(bs.sendChan) > SendChanSize/2 {
			fcDelay += bs.FCDelayIncreaseStep
		} else if len(bs.sendChan) < SendChanSize/3 && bs.fcDelay > 0 {
			// slow decrease for recovery
			fcDelay = fcDelay - bs.FCDelayDecreaseStep
		}
		bs.fcDelay = fcDelay
		if fcDelay > 0 {
			// slog.Info("- Send", "txMsgCnt", bs.txMsgCnt.Load(), "retrycnt", retryCount, "delay", bs.fcDelay,
			// "chan level", len(bs.sendChan))
			time.Sleep(fcDelay)
		}
		bs.mux.Unlock()
		select {
		case bs.sendChan <- rawMsg:
			// all is well
			cancelFn()
			return nil
		case <-ctx.Done():
			// context was closed. We're done here
			cancelFn()
			return nil
		default:
			if retryCount >= bs.MaxRetryCount {
				// ideally this never happens
				slog.Error("Failed to send. Client too slow.",
					"retryCount", retryCount, "delay", bs.fcDelay)
				return ErrClientTooSlow
			}
			// if the channel is full then retry, but double the delay
			bs.mux.Lock()
			bs.fcDelay = bs.fcDelay * 2
			fcDelay := bs.fcDelay
			bs.mux.Unlock()
			slog.Warn("Send: channel is full. Retrying and increasing send delay",
				"sendCnt", bs.txMsgCnt.Load(), "delay", fcDelay, "retryCount", retryCount)
		}
	}
}

// WaitUntilDisconnect waits until the send or receive stream is closed.
// Intended to be called by the (server) serve handler to avoid the stream from
// closing on return.
func (bs *BufferedStream) WaitUntilDisconnect() {
	if bs.isConnected.Load() {
		<-(bs.cancelChan)
		bs.isConnected.Store(false)
	}
}

// NewBufferedStream creates a concurrently safe buffered stream for a gRPC stream.
//
// This supports sending messages concurrently.
//
// If the send buffer is full, the client is considered too slow and the stream is closed
// after waiting the sendTimeout duration for the buffer to be available.
//
// Call WaitUntilDisconnect() after creating this instance to wait until the stream
// is closed by the client or the server. Needed by the server serve handler.
//
// The buffer Close() method will also call the provided stream context cancel method.
//
//	msgStream is the raw grpc message stream
//	cancelFn is the function to cancel the message stream context. nil if not applicable
//	recvHandler is called when a new message is received from the stream.
//	sendTimeout is the default timeout for sending messages on this stream
//	 when the send buffer is full.
func NewBufferedStream(
	msgStream IMsgStream, cancelFn func(), recvHandler func(rawMsg []byte), sendTimeout time.Duration,
) *BufferedStream {
	strm := &BufferedStream{
		msgStreamCancel: cancelFn,
		sendTimeout:     sendTimeout,
		msgStream:       msgStream,

		MaxRetryCount: 10,
		// flow control
		FCDelayDecreaseStep: FCDelayDecreaseStep,
		FCDelayIncreaseStep: FCDelayIncreaseStep,

		// ServiceMsgChanSize is the default buffer size.
		sendChan:   make(chan []byte, SendChanSize),
		cancelChan: make(chan struct{}),
	}
	strm.isConnected.Store(true)
	go strm._recvLoop(recvHandler)
	go strm._sendLoop()

	return strm
}
