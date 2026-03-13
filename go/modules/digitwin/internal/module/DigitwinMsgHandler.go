package module

import (
	"fmt"

	digitwinapi "github.com/hiveot/hivekit/go/modules/digitwin/api"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
)

// The handler for messages aimed at this module
type DigitwinMsgHandler struct {
	digitwinModule digitwinapi.IDigitwinModule
}

// HandleRequest handles requests aimed at the digital twin module
func (handler *DigitwinMsgHandler) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var resp *msg.ResponseMessage

	// Handle requests for this module

	// todo: add methods for digital twin status queries
	switch req.Operation {
	// case wot.OpReadProperty:
	// case wot.OpReadAllProperties:
	// case wot.OpReadMultipleProperties:
	// case wot.HTOpReadEvent:
	// case wot.HTOpReadAllEvents:
	case wot.OpWriteProperty:
		// nothing to do here at the moment
		err = fmt.Errorf("Property '%s' of Thing '%s' is invalid or not writable", req.Name, req.ThingID)
	case wot.OpInvokeAction:
		// directory specific operations
		switch req.Name {
		// case digitwinapi.GetStatusMethod:
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

func NewDigitwinMsgHandler(digitwinModule digitwinapi.IDigitwinModule) *DigitwinMsgHandler {
	handler := &DigitwinMsgHandler{
		digitwinModule: digitwinModule,
	}
	return handler
}
