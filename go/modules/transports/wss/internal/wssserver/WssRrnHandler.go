package wssserver

import (
	"fmt"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
)

// WssRrnHandler maps between RRN messages and the native module API.
type WssRrnHandler struct {
	module transports.ITransportServer
}

// HandleRequest for writing configuration and invoking module actions
// This has nothing to do with handling websocket requests.
//
// The request must be valid for this module before passing it.
// This returns an error if the request is not handled here.
func (msgAPI *WssRrnHandler) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	err = fmt.Errorf("Request '%s' not supported", req.Operation)

	// TODO: implement support for module actions or configuration
	return err
}

// Create a new instance of the RRN messaging API for the websocket module
func NewWssRrnHandler(module transports.ITransportServer) *WssRrnHandler {
	msgAPI := &WssRrnHandler{
		module: module,
	}
	return msgAPI
}
