package router

import (
	"errors"

	"github.com/hiveot/hivekitgo/messaging"
)

// MessageRouter diverts requests to the appropriate agent and returns responses to the sender.

type MessageRouter struct {
}

// HandleSyncRequest passes the message to the corresponding agent and wait for a response.
// If the request is synchronous then wait for a response.
func (router *MessageRouter) HandleSyncRequest(req *messaging.RequestMessage) (*messaging.ResponseMessage, error) {
	//
	return nil, errors.New("not yet implemented")
}

// HandleAsyncRequest passes the message to the corresponding agent and immediately returns
// When a response is received it is send asynchronously to the sender identified in the request.
func (router *MessageRouter) HandleAsyncRequest(req *messaging.RequestMessage) error {
	//
	return errors.New("not yet implemented")
}
