package api

import (
	"fmt"

	"github.com/hiveot/hivekit/go/modules/messaging"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// Embed the module TM - TODO: currently this module does not have a TM
//
// //go:embed httpbasic-tm.json
// var HttpBasicTMJson []byte

// HttpBasicMsgHandler maps SME messages to the native service interface.
type HttpBasicMsgHandler struct {
	// the module instance ThingID that must match the requests
	thingID string
	service transports.ITransportModule
}

// HandleRequest for writing configuration and invoking module actions
func (handler *HttpBasicMsgHandler) HandleRequest(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
	if req.ThingID != handler.thingID {
		return nil
	} else if req.SenderID == "" {
		err := fmt.Errorf("missing senderID in request")
		return req.CreateErrorResponse(err)
	}
	// nothing to do here. Reading properties is handled by the module base
	// at the moment there are no configurable properties in this module
	return nil
}

// Create a new instance of the HTTP-Basic transport
func NewHttpBasicMsgHandler(thingID string, service transports.ITransportModule) *HttpBasicMsgHandler {
	handler := &HttpBasicMsgHandler{
		thingID: thingID,
		service: service,
	}
	return handler
}
