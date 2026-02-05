package msg

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// the default timeout to use if none is provided
const DefaultRnRTimeout = time.Second * 3

// RnRChan is a helper for Request 'n Response message handling using channels.
// Intended to link responses in asynchronous request-response communication.
// This uses the correlationID to match responses to requests.
//
// The uses generics to specify the response message type.
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
	correlData map[string]chan *ResponseMessage
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
	rnr.correlData = make(map[string]chan *ResponseMessage)

}

// HandleResponse writes a reply to the request channel.
//
// This returns true on success or false if correlationID is unknown (no-one is waiting)
// It is up to the handler of this response to close the channel when done.
//
// If a timeout occurs while writing the response to the channel, the channel is full.
// This should never happen as the channel has a buffer of 1 and is removed after it is
// read, unless the channel isn't read while duplicate responses are received in which case
// this call fails with a timeout.
//
//	resp is the response message to send
//	timeout is the maximum time to wait for the RNR send channel to accept the message. 0 for default
func (rnr *RnRChan) HandleResponse(resp *ResponseMessage, timeout time.Duration) bool {

	if timeout == 0 {
		timeout = DefaultRnRTimeout
	}
	// Note: avoid a race between closing the channel and writing multiple responses.
	// This would happen if a 'pending' response arrives after a 'completed' response,
	// and 'wait-for-response' closes the channel while the second result is written.
	// This would panic, so lock the lookup and writing of the response channel.
	rnr.mux.Lock()
	rChan, isRPC := rnr.correlData[resp.CorrelationID]
	defer rnr.mux.Unlock()
	if isRPC {
		slog.Debug("HandleResponse: writing response to RnR go channel. ",
			slog.String("correlationID", resp.CorrelationID),
		)
		ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
		select {
		case rChan <- resp:
		case <-ctx.Done():
			// this should never happen
			slog.Error("Response RnR go channel is full. Is no-one listening?")
			// recover
			isRPC = false
		}
		cancelFn()
	} else {
		slog.Debug("HandleResponse: not an RnR call (subscription).",
			slog.String("correlationID", resp.CorrelationID))
	}
	return isRPC
}

func (rnr *RnRChan) Len() int {
	rnr.mux.Lock()
	defer rnr.mux.Unlock()
	return len(rnr.correlData)
}

// Open a channel for receiving response to a request with the given correlationID.
//
// This MUST be followed with one of these ways to close the channel:
//  1. WaitForResponse(correlationID)
//  2. WaitWithCallback(correlationID)
//  3. Close(correlationID)
//
// This returns a reply channel on which the data is received. Use
// WaitForResponse(rChan)
//
// If the correlationID already exists then this logs an error and panics.
func (rnr *RnRChan) Open(correlationID string) {
	//slog.Info("opening channel. ", "correlationID", correlationID)
	// this needs to be able to buffer 1 response in case completed and pending
	// are received out of order.
	rnr.mux.Lock()
	defer rnr.mux.Unlock()
	if _, found := rnr.correlData[correlationID]; found {
		// Attempt to recover but a correlationID should never be used twice at the same time.
		err := fmt.Errorf("RnRChan.Open: correlationID '%s' already exists. Recovered by returning its channel.", correlationID)
		slog.Error(err.Error())
		panic(err.Error())
	}
	rChan := make(chan *ResponseMessage, 1)
	rnr.correlData[correlationID] = rChan
}

// WaitForResponse blocks and waits for an answer received on the reply channel.
// This closes the channel on return.
//
// If a timeout occurs then this returns with 'hasResponse' false.
//
// If the correlationID does not exist, then this logs an error and returns with
// 'hasResponse' false.
func (rnr *RnRChan) WaitForResponse(
	correlationID string, timeout time.Duration) (hasResponse bool, resp *ResponseMessage) {

	if timeout == 0 {
		timeout = DefaultRnRTimeout
	}
	rnr.mux.RLock()
	replyChan, found := rnr.correlData[correlationID]
	rnr.mux.RUnlock()
	if !found {
		slog.Error("WaitForResponse: channel does not exist", "correlationID", correlationID)
		return false, nil
	}
	// good, a channel was previously opened
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	select {
	case rData := <-replyChan:
		resp = rData
		hasResponse = true
		break
	case <-ctx.Done(): // timeout
		hasResponse = false
	}
	cancelFunc()
	rnr.Close(correlationID)
	return hasResponse, resp
}

// WaitWithCallback listens for a response with the given correlationID and calls the
// handler when it is received. This returns immediately.
//
// If a channel with this correlationID already exists then it is used. This is intentional
// to allow creating a channel, sending the request and wait with callback after sending is
// successful.
//
// If a channel with this correlationID does NOT exist then one is created. This is also
// intentional to sending a request after WaitWithCallback.
//
// Note that if send failed the channel must be closed with Close() to end the WaitWithCallback.
//
// The channel is automatically closed on response or timeout.
//
// This immediately returns while waiting in the background.
// If a timeout occurs an error is logged
func (rnr *RnRChan) WaitWithCallback(correlationID string, timeout time.Duration, handler func(msg *ResponseMessage) error) {

	if timeout == 0 {
		timeout = DefaultRnRTimeout
	}
	// If a correlationID already exists then use its channel
	rnr.mux.RLock()
	_, found := rnr.correlData[correlationID]
	rnr.mux.RUnlock()
	if !found {
		// open a new channel
		rnr.Open(correlationID)
	}

	go func() {
		hasResponse, resp := rnr.WaitForResponse(correlationID, timeout)
		if hasResponse {
			_ = handler(resp)
		} else {
			slog.Error("RnrChan:WaitWithCallback. Timeout waiting for response",
				"timeout", timeout,
				"correlationID", correlationID)
		}
	}()
}

// Create a new instance of a Request & Response channel handler.
// This supports multiple concurrent requests. Responses are matched using the correlationID.
// Use WaitWithCallback or WaitForResponse to obtain the response.
func NewRnRChan() *RnRChan {
	r := &RnRChan{
		correlData: make(map[string]chan *ResponseMessage),
	}
	return r
}
