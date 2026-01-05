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
	err   error
	rChan chan T
}

func (rnr *AsyncReceiver[T]) WaitForResponse(timeout time.Duration) (T, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()

	select {
	case rnr.data = <-rnr.rChan:
		break
	case <-ctx.Done():
		rnr.err = errors.New("timeout")
	}
	return rnr.data, rnr.err
}

// Write the answer to the channel
func (rnr *AsyncReceiver[T]) SetResponse(data T, err error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second)
	select {
	case rnr.rChan <- data:
		rnr.err = err
	case <-ctx.Done():
		// this should never happen
		rnr.err = errors.New("Response RPC go channel is full. Is no-one listening?")
	}
	cancelFn()
}

func NewAsyncReceiver[T comparable]() AsyncReceiver[T] {
	r := AsyncReceiver[T]{
		// use a buffer of 1 to allow setting response before waiting
		rChan: make(chan T, 1),
	}
	return r
}
