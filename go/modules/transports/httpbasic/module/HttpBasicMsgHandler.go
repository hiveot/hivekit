package module

import (
	"fmt"

	"github.com/hiveot/hivekit/go/modules/transports/httpbasic"
	"github.com/hiveot/hivekit/go/msg"
)

// Embed the module TM - TODO: currently this module does not have a TM
//
// //go:embed httpbasic-tm.json
// var HttpBasicTMJson []byte

// HttpBasicMsgHandler maps between RRN messages and the native module interface.
type HttpBasicMsgHandler struct {
	module httpbasic.IHttpBasicTransport
}

// HandleRequest for writing configuration and invoking module actions
// This has nothing to do with handling http requests.
//
// The request must be valid for this module before passing it.
// This returns an error if the request is not handled here.
func (msgHandler *HttpBasicMsgHandler) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	err = fmt.Errorf("Request '%s' not supported", req.Operation)
	// TODO: implement support for module actions or configuration
	return err
}

// Create a new instance of the HTTP-Basic module RRN messaging handler
func NewHttpBasicMsgHandler(module httpbasic.IHttpBasicTransport) *HttpBasicMsgHandler {
	msgAPI := &HttpBasicMsgHandler{
		module: module,
	}
	return msgAPI
}
