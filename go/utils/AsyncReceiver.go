package utils

import (
	"context"
	"errors"
	"time"
)

// AsyncReceiver is a simple helper for waiting on data that will be received
// asynchronously.
//
// This supports passing an error as part of the response and setting a timeout
// to wait for the response.
//
// Usage is simple, call WaitForResponse with a timeout, and if a response is
// received asynchronously then call SetResponse.
type AsyncReceiver[T comparable] struct {
	data  T
	rChan chan T
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
// Returns the data and error set by SetResponse, or a timeout error.
func (arx *AsyncReceiver[T]) WaitForResponse(timeout time.Duration) (T, error) {
	var err error
	if timeout <= 0 {
		timeout = time.Second * 60
	}
	// create a context with timeout
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()

	select {
	case arx.data = <-arx.rChan:
		break
	case <-ctx.Done():
		err = errors.New("timeout")
	}
	return arx.data, err
}

func NewAsyncReceiver[T comparable]() AsyncReceiver[T] {
	r := AsyncReceiver[T]{
		// use a buffer of 1 to allow setting response before waiting
		rChan: make(chan T, 1),
	}
	return r
}
