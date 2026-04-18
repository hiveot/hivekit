package internal

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/digitwin"
)

// The handler for messages aimed at this module
type DigitwinMsgHandler struct {
	digitwinModule digitwin.IDigitwinService
}

// HandleRequest handles requests aimed at the digital twin module
func (handler *DigitwinMsgHandler) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var resp *msg.ResponseMessage

	// Handle requests for this module

	// todo: add methods for digital twin status queries
	switch req.Operation {
	// case td.OpReadProperty:
	// case td.OpReadAllProperties:
	// case td.OpReadMultipleProperties:
	// case td.HTOpReadEvent:
	// case td.HTOpReadAllEvents:
	case td.OpWriteProperty:
		// nothing to do here at the moment
		err = fmt.Errorf("Property '%s' of Thing '%s' is invalid or not writable", req.Name, req.ThingID)
	case td.OpInvokeAction:
		// directory specific operations
		switch req.Name {
		// case digitwin.GetStatusMethod:
		// 	resp = handler.HandleGetStatus(req)
		default:
			err = fmt.Errorf("Unknown request name '%s' for thingID '%s'", req.Name, req.ThingID)
			resp = nil
		}
	default:
		err = fmt.Errorf("Unsupported operation '%s' for thingID '%s'", req.Operation, req.ThingID)
		resp = nil
	}
	if resp != nil {
		err = replyTo(resp)
	} else if err == nil {
		err = fmt.Errorf("Unhandled request")
	}
	return err
}

func NewDigitwinMsgHandler(digitwinModule digitwin.IDigitwinService) *DigitwinMsgHandler {
	handler := &DigitwinMsgHandler{
		digitwinModule: digitwinModule,
	}
	return handler
}
