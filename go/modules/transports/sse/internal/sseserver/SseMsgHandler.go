package sseserver

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api/msg"
	sseapi "github.com/hiveot/hivekit/go/modules/transports/sse/api"
)

// HiveotSseMsgHandler maps between RRN messages and the native module interface.
type HiveotSseMsgHandler struct {
	// the module instance ThingID that must match the requests
	module sseapi.ISseTransportServer
}

// HandleRequest for writing configuration and invoking module actions
// This has nothing to do with handling http/sse requests.
//
// The request must be valid for this module before passing it.
// This returns an error if the request is not handled here.
func (msghandler *HiveotSseMsgHandler) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	err = fmt.Errorf("Request '%s' not supported", req.Operation)

	// TODO: implement support for module actions or configuration
	return err
}

// Create a new instance of the HTTP-Basic transport
func NewHiveotSseMsgHandler(module sseapi.ISseTransportServer) *HiveotSseMsgHandler {
	handler := &HiveotSseMsgHandler{
		module: module,
	}
	return handler
}
