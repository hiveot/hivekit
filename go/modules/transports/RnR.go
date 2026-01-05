package transports

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/msg"
)

// RnRChan is a helper for Request 'n Response message handling using channels.
// Intended to link responses in asynchronous request-response communication.
// This uses the correlationID to match responses to requests.
//
// Usage:
//  1. create a correlationID: shortid.MustGenerate()
//  2. register the correlationID: c := Open(correlationID)
//  3. Send the request message in the client, including the correlationID
//  4. Pass all responses to the RnRChan HandlerResponse handler
//     This returns handled==false if the response must be handled manually.
//     This returns handled==true if the response was passed to a waiting handler (step 5)
//  5. Wait for a matching response using:
//     A: WaitForResponse which will block until a response is received
//     B: WaitWithCallback which returns immediately and invokes the callback when a response is received
type RnRChan struct {
	mux sync.RWMutex

	// map of correlationID to delivery status update channel
	correlData map[string]chan *msg.ResponseMessage

	//timeout write to a response channel
	writeTimeout time.Duration
}

// Close removes the request channel
func (rnr *RnRChan) Close(correlationID string) {
	rnr.mux.Lock()
	defer rnr.mux.Unlock()

	//slog.Info("closing channel. ", "correlationID", correlationID)
	rChan, found := rnr.correlData[correlationID]
	if found {
		delete(rnr.correlData, correlationID)
		close(rChan)
	}
}

// CloseAll force closes all channels, ending all waits for RPC responses.
func (rnr *RnRChan) CloseAll() {
	rnr.mux.Lock()
	defer rnr.mux.Unlock()
	for _, rChan := range rnr.correlData {
		close(rChan)
	}
	rnr.correlData = make(map[string]chan *msg.ResponseMessage)

}

// HandleResponse writes a reply to the request channel.
//
// This returns true on success or false if correlationID is unknown (no-one is waiting)
// It is up to the handler of this response to close the channel when done.
//
// If a timeout passes while writing is block the write is released.
func (rnr *RnRChan) HandleResponse(msg *msg.ResponseMessage) bool {
	// Note: avoid a race between closing the channel and writing multiple responses.
	// This would happen if a 'pending' response arrives after a 'completed' response,
	// and 'wait-for-response' closes the channel while the second result is written.
	// This would panic, so lock the lookup and writing of the response channel.
	rnr.mux.Lock()
	rChan, isRPC := rnr.correlData[msg.CorrelationID]
	defer rnr.mux.Unlock()
	if isRPC {
		slog.Debug("HandleResponse: writing response to RPC go channel. ",
			slog.String("correlationID", msg.CorrelationID),
			slog.String("operation", msg.Operation),
		)
		ctx, cancelFn := context.WithTimeout(context.Background(), rnr.writeTimeout)
		select {
		case rChan <- msg:
		case <-ctx.Done():
			// this should never happen
			slog.Error("Response RPC go channel is full. Is no-one listening?")
			// recover
			isRPC = false
		}
		cancelFn()
	} else {
		//slog.Info("HandleResponse: not an RPC call (subscription).",
		//	slog.String("correlationID", msg.CorrelationID),
		//	slog.String("operation", msg.Operation))
	}
	return isRPC
}

func (rnr *RnRChan) Len() int {
	rnr.mux.Lock()
	defer rnr.mux.Unlock()
	return len(rnr.correlData)
}

// Open a new channel for receiving response to a request
// Call Close(correlationID) when done.
//
// This returns a reply channel on which the data is received. Use
// WaitForResponse(rChan)
func (rnr *RnRChan) Open(correlationID string) chan *msg.ResponseMessage {
	//slog.Info("opening channel. ", "correlationID", correlationID)
	// this needs to be able to buffer 1 response in case completed and pending
	// are received out of order.
	rChan := make(chan *msg.ResponseMessage, 1)
	rnr.mux.Lock()
	rnr.correlData[correlationID] = rChan
	rnr.mux.Unlock()
	return rChan
}

// WaitForResponse waits for an answer received on the reply channel.
// After timeout without response this returns with completed is false.
// if timeout is 0, the default 60 second timeout is used
//
// If the channel was closed this returns hasResponse with no reply
func (rnr *RnRChan) WaitForResponse(
	replyChan chan *msg.ResponseMessage, timeout time.Duration) (
	hasResponse bool, resp *msg.ResponseMessage) {

	if timeout == 0 {
		timeout = time.Second * 60
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()
	select {
	case rData := <-replyChan:
		resp = rData
		hasResponse = true
		break
	case <-ctx.Done():
		hasResponse = false
	}
	return hasResponse, resp
}

// WaitWithCallback opens a new channel for receiving responses via a
// callback handler in the background.
// The channel is automatically closed on response or timeout.
//
// WaitWithCallback must be invoked before sending the request to ensure that
// an immediate response is captured.
//
// This immediately returns while waiting in the background.
// If a timeout occurs an error is logged
func (rnr *RnRChan) WaitWithCallback(
	correlationID string, handler msg.ResponseHandler, timeout time.Duration) {
	rChan := rnr.Open(correlationID)
	go func() {
		hasResponse, resp := rnr.WaitForResponse(rChan, timeout)
		rnr.Close(correlationID)
		if hasResponse {
			_ = handler(resp)
		} else {
			slog.Error("RnrChan:WaitWithCallback. Timeout waiting for response",
				"timeout", timeout/time.Second,
				"correlationID", correlationID)
		}
	}()
}
func NewRnRChan() *RnRChan {
	r := &RnRChan{
		correlData:   make(map[string]chan *msg.ResponseMessage),
		writeTimeout: time.Second * 300, // default 3
	}
	return r
}
