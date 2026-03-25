package server

import (
	"fmt"

	httpbasicapi "github.com/hiveot/hivekit/go/modules/transports/httpbasic/api"
	"github.com/hiveot/hivekit/go/msg"
)

// Embed the module TM - TODO: currently this module does not have a TM
//
// //go:embed httpbasic-tm.json
// var HttpBasicTMJson []byte

// HttpBasicMsgHandler maps between RRN messages and the native module interface.
type HttpBasicMsgHandler struct {
	module httpbasicapi.IHttpBasicServer
}

// HandleRequest for this module's requests.
// This has nothing to do with handling http requests.
//
// The request must be valid for this module before passing it.
// This returns an error if the request is not handled here.
func (msgHandler *HttpBasicMsgHandler) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// this should only be invoked if the thingID is the moduleID

	err = fmt.Errorf("Request '%s' not supported", req.Operation)
	// TODO: implement support for module actions or configuration
	return err
}

// Create a new instance of the HTTP-Basic module RRN messaging handler
func NewHttpBasicMsgHandler(module httpbasicapi.IHttpBasicServer) *HttpBasicMsgHandler {
	msgAPI := &HttpBasicMsgHandler{
		module: module,
	}
	return msgAPI
}
