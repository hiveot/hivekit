package utils

import (
	"context"
	"errors"
	"time"
)

// AsyncReceiver is a simple helper for waiting on data that will be received
// asynchronously.
//
// Usage is simple, call WaitForResponse, and if a response is
// received asynchronously then call SetResponse.
type AsyncReceiver[T comparable] struct {
	data    T
	rChan   chan T
	timeout time.Duration
}

// Cancel the channel. Use this instead of SetResponse if no response is avaialble.
func (arx *AsyncReceiver[T]) Cancel(data T) {
	close(arx.rChan)
}

// Write the answer to the channel
func (arx *AsyncReceiver[T]) SetResponse(data T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), arx.timeout)
	select {
	case arx.rChan <- data:
	case <-ctx.Done():
		// this should never happen
		panic("Response RPC go channel is full. Is no-one listening?")
	}
	cancelFn()
}

// WaitForResponse waits for the response to be set or times out.
//
// Returns the data set by SetResponse, or an error on timeout or cancel.
func (arx *AsyncReceiver[T]) WaitForResponse() (T, error) {
	var err error
	var ok bool

	// create a context with timeout
	ctx, cancelFunc := context.WithTimeout(context.Background(), arx.timeout)
	defer cancelFunc()

	select {
	case arx.data, ok = <-arx.rChan:
		if !ok {
			err = errors.New("WaitForResponse: Request was cancelled")
		}
		break
	case <-ctx.Done():
		err = errors.New("WaitForResponse: timeout")
	}
	return arx.data, err
}

// Create a new receiver of async messages.
// TODO: this should take a context that can be cancelled.
func NewAsyncReceiver[T comparable](timeout time.Duration) AsyncReceiver[T] {
	r := AsyncReceiver[T]{
		// use a buffer of 1 to allow setting response before waiting
		rChan:   make(chan T, 1),
		timeout: timeout,
	}
	return r
}
