package module

import (
	_ "embed"
	"fmt"

	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
)

// Embed the directory TM
//
//go:embed "directory-tm.json"
var DirectoryTMJson []byte

// DirectoryMsgHandler maps RRN messages to the native directory interface
type DirectoryMsgHandler struct {
	// the directory instance ThingID that must match the requests
	thingID string
	service directoryapi.IDirectoryModuleServer
}

// GetTm returns the TN of the directory RRN messaging API
func (handler *DirectoryMsgHandler) GetTM() string {
	tm := string(DirectoryTMJson)
	return tm
}

// HandleRequest for module.
//
// This invokes the replyTo response handler with a response.
//
// If the request is not for this module then it is forwarded to the next sink.
// If the request is for this module but invalid, an error is returned
func (handler *DirectoryMsgHandler) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var resp *msg.ResponseMessage
	if req.ThingID != handler.thingID {
		return nil
	} else if req.SenderID == "" {
		err := fmt.Errorf("missing senderID in request")
		return err
	}
	if req.Operation == wot.OpInvokeAction {
		// directory specific operations
		switch req.Name {
		case directoryapi.ActionCreateThing:
			resp = handler.UpdateThing(req)
		case directoryapi.ActionDeleteThing:
			resp = handler.DeleteThing(req)
		case directoryapi.ActionRetrieveThing:
			resp = handler.RetrieveThing(req)
		case directoryapi.ActionRetrieveAllThings:
			resp = handler.RetrieveAllThings(req)
		case directoryapi.ActionUpdateThing:
			resp = handler.UpdateThing(req)
		default:
			err = fmt.Errorf("Unknown request name '%s' for thingID '%s'", req.Name, req.ThingID)
		}
	} else if req.Operation == wot.OpWriteProperty {
		// nothing to do here at the moment
		err = fmt.Errorf("Property '%s' of Thing '%s' is invalid or not writable", req.Name, req.ThingID)
	} else {
		err = fmt.Errorf("Unsupported operation '%s' for thingID '%s'", req.Operation, req.ThingID)
	}
	if resp != nil {
		err = replyTo(resp)
	}
	return err
}

// DeleteThing removes a thing in the directory
// req.Input is a string containing the Thing ID
func (handler *DirectoryMsgHandler) DeleteThing(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	var thingID string
	err := utils.Decode(req.Input, &thingID)
	if err == nil {
		err = handler.service.DeleteThing(req.SenderID, thingID)
	}
	resp = req.CreateResponse(nil, err)
	return resp
}

// RetrieveAllThings returns a list of things
// Input: {offset, limit}
func (handler *DirectoryMsgHandler) RetrieveAllThings(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	var tdList []string
	var err error
	var args directoryapi.RetrieveAllThingsArgs

	err = utils.Decode(req.Input, &args)
	if err == nil {
		tdList, err = handler.service.RetrieveAllThings(args.Offset, args.Limit)
	}
	resp = req.CreateResponse(tdList, err)
	return resp
}

// RetrieveThing gets the TD JSON for the given thingID from the directory store.
func (handler *DirectoryMsgHandler) RetrieveThing(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	var thingID string
	var tdJSON string
	err := utils.Decode(req.Input, &thingID)
	if err == nil {
		tdJSON, err = handler.service.RetrieveThing(thingID)
	}
	resp = req.CreateResponse(tdJSON, err)
	return resp
}

// UpdateThing updates a new thing in the store
// req.Input is a string containing the TD JSON
//
// Requirement: for security reasons only the agent that owns the TD is allowed to update it
func (handler *DirectoryMsgHandler) UpdateThing(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	var tdJSON string

	err := utils.Decode(req.Input, &tdJSON)
	if err == nil {
		err = handler.service.UpdateThing(req.SenderID, tdJSON)
	}
	resp = req.CreateResponse(nil, err)
	return resp
}

// Create a new directory message handler. On start this creates the server and store.
// bucketStore is the store to use for this module chain.
func NewDirectoryMsgHandler(thingID string, store directoryapi.IDirectoryModuleServer) *DirectoryMsgHandler {

	handler := &DirectoryMsgHandler{
		thingID: thingID,
		service: store,
	}
	return handler
}
