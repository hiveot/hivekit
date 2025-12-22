package api

import (
	_ "embed"
	"fmt"

	"github.com/hiveot/hivekit/go/modules/services/directory"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	jsoniter "github.com/json-iterator/go"
)

// Embed the directory TM
//
//go:embed directory-tm.json
var DirectoryTMJson []byte

// DirectoryMsgHandler maps RRN messages to the native directory interface
type DirectoryMsgHandler struct {
	// the directory instance ThingID that must match the requests
	thingID string
	service directory.IDirectoryService
}

// HandleRequest for properties or actions
// If the request is not recognized nil is returned.
// If the request is missing the sender, an error is returned
func (handler *DirectoryMsgHandler) HandleRequest(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	if req.ThingID != handler.thingID {
		return nil
	} else if req.SenderID == "" {
		err := fmt.Errorf("missing senderID in request")
		return req.CreateErrorResponse(err)
	}
	if req.Operation == wot.OpInvokeAction {
		// directory specific operations
		switch req.Name {
		case ActionCreateThing:
			resp = handler.UpdateThing(req)
		case ActionDeleteThing:
			resp = handler.DeleteThing(req)
		case ActionRetrieveThing:
			resp = handler.RetrieveThing(req)
		case ActionRetrieveAllThings:
			resp = handler.RetrieveAllThings(req)
		case ActionUpdateThing:
			resp = handler.UpdateThing(req)
		}
	}
	return resp
}

// DeleteThing removes a thing in the directory
// req.Input is a string containing the Thing ID
func (handler *DirectoryMsgHandler) DeleteThing(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	var thingID string
	err := utils.Decode(req.Input, &thingID)
	if err == nil {
		err = handler.service.DeleteThing(thingID)
	}
	resp = req.CreateResponse(nil, err)
	return resp
}

// GetTD this module returns the directory TD
// It includes forms for http access through the REST API.
func (handler *DirectoryMsgHandler) GetTD() (tdDoc *td.TD) {
	tdJson := DirectoryTMJson
	jsoniter.Unmarshal(tdJson, &tdDoc)

	return tdDoc
}

// RetrieveAllThings returns a list of things
// Input: {offset, limit}
func (handler *DirectoryMsgHandler) RetrieveAllThings(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	var tdList []string
	var err error
	var args RetrieveAllThingsArgs

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
func (handler *DirectoryMsgHandler) UpdateThing(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	var tdJSON string

	err := utils.Decode(req.Input, &tdJSON)
	if err == nil {
		err = handler.service.UpdateThing(tdJSON)
	}
	resp = req.CreateResponse(nil, err)
	return resp
}

// Create a new directory message handler. On start this creates the server and store.
// bucketStore is the store to use for this module chain.
func NewDirectoryMsgHandler(thingID string, store directory.IDirectoryService) *DirectoryMsgHandler {

	handler := &DirectoryMsgHandler{
		thingID: thingID,
		service: store,
	}
	return handler
}
