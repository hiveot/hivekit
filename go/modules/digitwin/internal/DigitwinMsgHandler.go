package internal

import (
	"fmt"
	"strings"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/digitwin"
)

// HandleRequest for digital twins requests.
// This looks at the dtw prefix to determine if this is a digital twin.
//
// - handle read requests directly from cache
// - route write requests to device
// - route action requests to device
//
// This invokes the replyTo response handler with a response.
//
// If the request is not for this module then it is forwarded to the next sink.
// If the request is for this module but invalid, an error is returned
func (m *DigitwinServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// Handle requests for a digital twin
	// TODO: try to remove the thingID dependency on the digital twin.
	// Maybe lookup the thingID in the digital twin directory... ?
	if strings.HasPrefix(req.ThingID, digitwin.DigitwinIDPrefix) {
		switch req.Operation {

		// read requests are handled by the value cache
		// the value cache holds digital twin property and event notifications

		// Note unobservable properties will never be in the cache. If the value isn't cached
		// the request is forwarded to the device. Eg set vcache sink to the server or client
		// connection that can forward it.
		case td.OpReadAllProperties,
			td.OpReadMultipleProperties,
			td.OpReadProperty, // this returns the property value
			td.HTOpReadEvent,  // this returns the event notification (not just the value)
			td.HTOpReadAllEvents:
			err = m.vcache.HandleRequest(req, replyTo)
			if err != nil {
				// vcache didn't handle the request, so forward it
				return m.ForwardDigitwinRequestToDevice(req, replyTo)
			}

		// write requests are forwarded to the actual device after mapping
		// the thingID back to that of the device
		case td.OpWriteProperty,
			td.OpWriteMultipleProperties,
			td.OpInvokeAction:

			return m.ForwardDigitwinRequestToDevice(req, replyTo)
		}
	}

	// Handle requests for this module
	if req.ThingID != m.GetThingID() {
		return nil
	} else if req.SenderID == "" {
		err := fmt.Errorf("missing senderID in request")
		return err
	}
	return m.HandleDigitwinRequest(req, replyTo)
}

// HandleRequest handles requests aimed at the digital twin module itself
func (handler *DigitwinServiceImpl) HandleDigitwinRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
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
