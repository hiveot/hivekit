package api

import (
	"errors"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/services/directory"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
)

// DirectoryMsgClient is a client for the Directory service using RRN messages.
// This implements the IDirectory interface.
// Intended to use a client transport module as sink, that forwards the messages.
type DirectoryMsgClient struct {
	modules.HiveModuleBase

	// DirectoryMsgClient is the RRN client for the directory service.

	// directoryID ThingID of the directory service. This defaults to the directory ThingID
	directoryID string
	// sink that forwards the messages
	// sink modules.IHiveModule

	// consumer *transports.Consumer
}

func (cl *DirectoryMsgClient) CreateThing(tdJson string) error {
	req := msg.NewRequestMessage(wot.OpInvokeAction, cl.directoryID, ActionCreateThing, tdJson, "")
	_, err := cl.ForwardRequestWait(req)
	return err
}

func (cl *DirectoryMsgClient) DeleteThing(thingID string) error {
	req := msg.NewRequestMessage(wot.OpInvokeAction, cl.directoryID, ActionDeleteThing, thingID, "")
	_, err := cl.ForwardRequestWait(req)
	return err
}

func (cl *DirectoryMsgClient) RetrieveThing(thingID string) (tdJSON string, err error) {
	req := msg.NewRequestMessage(wot.OpInvokeAction, cl.directoryID, ActionRetrieveThing, thingID, "")
	resp, err := cl.ForwardRequestWait(req)
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
	req := msg.NewRequestMessage(wot.OpInvokeAction, cl.directoryID, ActionRetrieveAllThings, args, "")
	resp, err := cl.ForwardRequestWait(req)
	if err == nil {
		err = resp.Decode(&tdList)
	}
	return tdList, err
}

func (cl *DirectoryMsgClient) UpdateThing(tdJson string) error {
	req := msg.NewRequestMessage(
		wot.OpInvokeAction, cl.directoryID, ActionUpdateThing, tdJson, "")
	_, err := cl.ForwardRequestWait(req)
	return err
}

// NewDirectoryMsgClient creates a new DirectoryMsgClient instance.
// Use the sink to attach a transport module
//
//	thingID is the unique ID of the directory service instance. This defaults to the directory module's thingID.
//	sink is the handler of request messages
func NewDirectoryMsgClient(thingID string, sink modules.IHiveModule) *DirectoryMsgClient {
	if thingID == "" {
		thingID = directory.DefaultDirectoryThingID
	}
	client := &DirectoryMsgClient{
		directoryID: thingID,
	}
	client.Init(thingID+"-client", sink)
	return client
}
