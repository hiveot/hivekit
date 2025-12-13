package api

import (
	"errors"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/messaging"
	"github.com/hiveot/hivekit/go/modules/services/directory"
	"github.com/hiveot/hivekit/go/wot"
)

// DirectoryMsgClient is a client for the Directory service using SME messages.
// This implements the IDirectory interface.
type DirectoryMsgClient struct {
	// DirectoryMsgClient is the SME client for the directory service.

	// directoryID ThingID of the directory service. This defaults to the directory ThingID
	directoryID string
	// sink that forwards the messages
	sink modules.IHiveModule
}

func (cl *DirectoryMsgClient) CreateThing(tdJson string) error {
	req := messaging.NewRequestMessage(
		wot.OpInvokeAction, cl.directoryID, ActionCreateThing, tdJson, "")
	resp := cl.sink.HandleRequest(req)
	return resp.AsError()
}

func (cl *DirectoryMsgClient) DeleteThing(thingID string) error {
	req := messaging.NewRequestMessage(
		wot.OpInvokeAction, cl.directoryID, ActionDeleteThing, thingID, "")
	resp := cl.sink.HandleRequest(req)
	return resp.AsError()
}

func (cl *DirectoryMsgClient) RetrieveThing(thingID string) (tdJSON string, err error) {
	req := messaging.NewRequestMessage(
		wot.OpInvokeAction, cl.directoryID, ActionRetrieveThing, thingID, "")
	resp := cl.sink.HandleRequest(req)
	if resp == nil {
		return "", errors.New("nil response")
	}
	if err = resp.AsError(); err == nil {
		err = resp.Decode(&tdJSON)
	}
	return tdJSON, err
}

func (cl *DirectoryMsgClient) RetrieveAllThings(offset int, limit int) (tdList []string, err error) {
	args := RetrieveAllThingsArgs{
		Offset: offset,
		Limit:  limit,
	}
	req := messaging.NewRequestMessage(
		wot.OpInvokeAction, cl.directoryID, ActionRetrieveAllThings, args, "")
	resp := cl.sink.HandleRequest(req)
	if err = resp.AsError(); err == nil {
		err = resp.Decode(&tdList)
	}
	return tdList, err
}

func (cl *DirectoryMsgClient) UpdateThing(tdJson string) error {
	req := messaging.NewRequestMessage(
		wot.OpInvokeAction, cl.directoryID, ActionUpdateThing, tdJson, "")
	resp := cl.sink.HandleRequest(req)
	return resp.AsError()
}

// NewDirectoryMsgClient creates a new DirectoryMsgClient instance.
// Use AddSink to attach a transport module
//
//	thingID is the unique ID of the directory service instance. This defaults to the directory module's thingID.
//	sink is the handler of request messages
func NewDirectoryMsgClient(thingID string, sink modules.IHiveModule) *DirectoryMsgClient {
	if thingID == "" {
		thingID = directory.DefaultDirectoryThingID
	}
	client := &DirectoryMsgClient{
		directoryID: thingID,
		sink:        sink,
	}
	return client
}
