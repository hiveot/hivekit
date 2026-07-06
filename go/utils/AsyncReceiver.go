package utils

import (
	"context"
	"errors"
	"time"
)

// AsyncReceiver is a simple helper for waiting on data that will be received
// asynchronously.
//
// Usage is simple, call WaitForResponse with a timeout, and if a response is
// received asynchronously then call SetResponse.
type AsyncReceiver[T comparable] struct {
	data  T
	rChan chan T
}

// Cancel the channel. Use this instead of SetResponse if no response is avaialble.
func (arx *AsyncReceiver[T]) Cancel(data T) {
	close(arx.rChan)
}

// Write the answer to the channel
func (arx *AsyncReceiver[T]) SetResponse(data T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second)
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
// If timeout is 0 or negative, a default of 60 seconds is used.
//
// Returns the data set by SetResponse, or an error on timeout or cancel.
func (arx *AsyncReceiver[T]) WaitForResponse(timeout time.Duration) (T, error) {
	var err error
	var ok bool
	if timeout <= 0 {
		timeout = time.Second * 60
	}
	// create a context with timeout
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()

	select {
	case arx.data, ok = <-arx.rChan:
		if !ok {
			err = errors.New("Request was cancelled")
		}
		break
	case <-ctx.Done():
		err = errors.New("timeout")
	}
	return arx.data, err
}

// Create a new receiver of async messages.
// FIXME: this should take a context that can be cancelled.
func NewAsyncReceiver[T comparable]() AsyncReceiver[T] {
	r := AsyncReceiver[T]{
		// use a buffer of 1 to allow setting response before waiting
		rChan: make(chan T, 1),
	}
	return r
}
