package internal

import (
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/utils"
)

// HandleRequest for directory service.
//
// Always support the default directory thingID defined in directory.DefaultDirectoryServiceThingID.
// If a different thingID is configured this should still work. Especially
// if used in a gateway and a gateway connection exists the TDD shouldn't be
// neccesary.
//
// This invokes the replyTo response handler with a response.
//
// If the request is not for this service then it is forwarded to the next sink.
// If the request is for this service but invalid, an error is returned
func (svc *DirectoryServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var resp *msg.ResponseMessage

	myThingID := svc.GetID()
	// other requests not directed at the directory are forwarded
	if req.ThingID != myThingID && req.ThingID != directory.DirectoryServiceCellType {
		return svc.ForwardRequest(req, replyTo)
	}

	slog.Info("HandleRequest", slog.String("name", req.Name), slog.String("sender", req.SenderID))

	// this is a request addressed to this service
	switch req.Operation {
	case wot.OpInvokeAction:
		// directory specific operations
		switch req.Name {
		case directory.CreateThingAction:
			resp = svc.handleUpdateThing(req)
		case directory.DeleteThingAction:
			resp = svc.handleDeleteThing(req)
		case directory.RetrieveTDDAction:
			resp = svc.handleRetrieveTDD(req)
		case directory.RetrieveThingAction:
			resp = svc.handleRetrieveThing(req)
		case directory.RetrieveAllThingsAction:
			resp = svc.handleRetrieveAllThings(req)
		case directory.UpdateThingAction:
			resp = svc.handleUpdateThing(req)
		default:
			err = fmt.Errorf("Unknown request name '%s' for thingID '%s'", req.Name, req.ThingID)
		}
	case wot.OpWriteProperty:
		// nothing to do here at the moment
		err = fmt.Errorf("Property '%s' of Thing '%s' is invalid or not writable", req.Name, req.ThingID)
	default:
		err = svc.ExposedThing.HandleRequest(req, replyTo)
	}
	if resp != nil {
		slog.Info("HandleRequest - sending response", slog.String("name", req.Name), slog.String("sender", req.SenderID))
		err = replyTo(resp)
	} else if err == nil {
		slog.Error("HandleRequest - No response AND no error. This is unexpected!")
	} else {
		slog.Info("HandleRequest - returning error",
			slog.String("name", req.Name), slog.String("sender", req.SenderID),
			slog.String("err", err.Error()))
	}
	return err
}

// DeleteThing removes a thing in the directory
// req.Input is a string containing the Thing ID
func (svc *DirectoryServiceImpl) handleDeleteThing(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	var thingID string
	err := utils.Decode(req.Input, &thingID)
	if err == nil {
		err = svc.DeleteThing(req.SenderID, thingID)
	}
	resp = req.CreateResponse(nil, err)
	return resp
}

// RetrieveAllThings returns a list of things
// Input: {offset, limit}
func (svc *DirectoryServiceImpl) handleRetrieveAllThings(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	var tdList []string
	var err error
	var args directory.RetrieveAllThingsArgs

	err = utils.Decode(req.Input, &args)
	if err == nil {
		tdList, err = svc.RetrieveAllThings(args.Offset, args.Limit)
	}
	resp = req.CreateResponse(tdList, err)
	return resp
}

// Read the directory TDD itself.
// Intended for retrieving the TDD using RRN messaging
// Output: tddJSON
func (svc *DirectoryServiceImpl) handleRetrieveTDD(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	tdd := svc.GetTDD()
	tddJSON := tdd.ToJSON()
	resp = req.CreateResponse(tddJSON, nil)
	return resp
}

// RetrieveThing gets the TD JSON for the given thingID from the directory store.
func (svc *DirectoryServiceImpl) handleRetrieveThing(req *msg.RequestMessage) (resp *msg.ResponseMessage) {

	var thingID string
	var tdJSON string
	err := utils.Decode(req.Input, &thingID)
	if err == nil {
		tdJSON, err = svc.RetrieveThing(thingID)
	}
	resp = req.CreateResponse(tdJSON, err)
	return resp
}

// UpdateThing updates a new thing in the store
// req.Input is a string containing the TD JSON
//
// Requirement: for security reasons only the client that owns the TD is allowed to update it
func (svc *DirectoryServiceImpl) handleUpdateThing(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	var tdJSON string

	err := utils.Decode(req.Input, &tdJSON)
	if err == nil {
		err = svc.UpdateThing(req.SenderID, tdJSON)
	}
	resp = req.CreateResponse(nil, err)
	return resp
}
