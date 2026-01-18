package module

import (
	"fmt"

	"github.com/hiveot/hivekit/go/modules/transports/ssesc"
	"github.com/hiveot/hivekit/go/msg"
)

// SseScMsgHandler maps between RRN messages and the native module interface.
type SseScMsgHandler struct {
	// the module instance ThingID that must match the requests
	module ssesc.ISseScTransport
}

// HandleRequest for writing configuration and invoking module actions
// This has nothing to do with handling http/sse requests.
//
// The request must be valid for this module before passing it.
// This returns an error if the request is not handled here.
func (msghandler *SseScMsgHandler) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	err = fmt.Errorf("Request '%s' not supported", req.Operation)

	// TODO: implement support for module actions or configuration
	return err
}

// Create a new instance of the HTTP-Basic transport
func NewSseScMsgHandler(module ssesc.ISseScTransport) *SseScMsgHandler {
	handler := &SseScMsgHandler{
		module: module,
	}
	return handler
}
