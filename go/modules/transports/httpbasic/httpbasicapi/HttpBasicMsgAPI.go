package httpbasicapi

import (
	"fmt"

	"github.com/hiveot/hivekit/go/modules/transports/httpbasic"
	"github.com/hiveot/hivekit/go/msg"
)

// Embed the module TM - TODO: currently this module does not have a TM
//
// //go:embed httpbasic-tm.json
// var HttpBasicTMJson []byte

// HttpBasicMsgAPI maps between RRN messages and the native module interface.
type HttpBasicMsgAPI struct {
	module httpbasic.IHttpBasicTransport
}

// HandleRequest for writing configuration and invoking module actions
// This has nothing to do with handling http requests.
//
// The request must be valid for this module before passing it.
// This returns an error if the request is not handled here.
func (msgAPI *HttpBasicMsgAPI) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	err = fmt.Errorf("Request '%s' not supported", req.Operation)
	// TODO: implement support for module actions or configuration
	return err
}

// Create a new instance of the HTTP-Basic transport
func NewHttpBasicMsgAPI(module httpbasic.IHttpBasicTransport) *HttpBasicMsgAPI {
	msgAPI := &HttpBasicMsgAPI{
		module: module,
	}
	return msgAPI
}
