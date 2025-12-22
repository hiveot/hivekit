package api

import (
	"fmt"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
)

// Embed the module TM - TODO: currently this module does not have a TM
//
// //go:embed httpbasic-tm.json
// var HttpBasicTMJson []byte

// HttpBasicRRNHandler maps between RRN messages and the native service interface.
type HttpBasicRRNHandler struct {
	// the module instance ThingID that must match the requests
	thingID string
	service transports.ITransportModule
}

// HandleRequest for writing configuration and invoking module actions
func (handler *HttpBasicRRNHandler) HandleRequest(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
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
func NewHttpBasicMsgHandler(thingID string, service transports.ITransportModule) *HttpBasicRRNHandler {
	handler := &HttpBasicRRNHandler{
		thingID: thingID,
		service: service,
	}
	return handler
}
