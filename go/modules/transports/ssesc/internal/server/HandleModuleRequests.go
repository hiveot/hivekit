package server

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api/msg"
)

// HiveotSseScMsgHandler maps between RRN messages and the native module interface.
// type HiveotSseScMsgHandler struct {
// 	// the module instance ThingID that must match the requests
// 	module ssescapi.ISseScTransportServer
// }

// HandleModuleRequest for writing configuration and invoking module actions
// This has nothing to do with handling http/sse requests.
//
// The request must be valid for this module before passing it.
// This returns an error if the request is not handled here.
func (c *TransportServer) HandleModuleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	err = fmt.Errorf("Request '%s' not supported", req.Operation)

	// TODO: implement support for module actions or configuration
	return err
}

// // Create a new instance of the messaging request handler that handles requests for this server
// func NewHiveotSseMsgHandler(module ssescapi.ISseScTransportServer) *HiveotSseScMsgHandler {
// 	handler := &HiveotSseScMsgHandler{
// 		module: module,
// 	}
// 	return handler
// }
