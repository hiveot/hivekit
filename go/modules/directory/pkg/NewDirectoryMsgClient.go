package directorypkg

import (
	"errors"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
)

// DirectoryMsgClient is a client for the Directory service using RRN messages.
// This implements the IDirectory interface and accepts a messaging protocol sink.
// Intended to use a client transport module as sink, that forwards the messages.
type DirectoryMsgClient struct {
	modules.HiveModuleBase

	// DirectoryMsgClient is the RRN client for the directory service.

	// directoryThingID ThingID of the directory service. This defaults to the directory ThingID
	directoryThingID string
}

func (cl *DirectoryMsgClient) DeleteThing(thingID string) error {
	req := msg.NewRequestMessage(
		td.OpInvokeAction, cl.directoryThingID, directory.ActionDeleteThing, thingID, "")
	_, err := cl.ForwardRequestWait(req)
	return err
}

func (cl *DirectoryMsgClient) RetrieveThing(thingID string) (tdJSON string, err error) {
	req := msg.NewRequestMessage(
		td.OpInvokeAction, cl.directoryThingID, directory.ActionRetrieveThing, thingID, "")
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
	args := directory.RetrieveAllThingsArgs{
		Offset: offset,
		Limit:  limit,
	}
	req := msg.NewRequestMessage(
		td.OpInvokeAction, cl.directoryThingID, directory.ActionRetrieveAllThings, args, "")
	resp, err := cl.ForwardRequestWait(req)
	if err == nil {
		err = resp.Decode(&tdList)
	}
	return tdList, err
}

// Update a Thing TD in the directory and wait for confirmation
// This retuns nil if success or an error if something went wrong.
// func (cl *DirectoryMsgClient) UpdateTD(tdJson string) error {

// 	req := msg.NewRequestMessage(
// 		td.OpInvokeAction, cl.directoryID, directory.ActionUpdateThing, tdJson, "")
// 	_, err := cl.ForwardRequestWait(req)

// 	return err
// }

// NewDirectoryMsgClient creates a new DirectoryMsgClient instance for consumers.
// Use the sink to attach a transport module.
//
// Do not use this client with agents as it registers itself as the receiver of notifications.
// This would prevent the agent to send its notifications to the server. Instead, use
// the 'UpdateThing' method below.
//
// This registers the directory client as the sink for notifications from the request handler.
// with the intent to receive directory updates.
//
//	serviceID is the thing ID of the directory service instance. This defaults to the directory module's type.
//	reqSink is the handler for requests send by the directory client and emitter of notifications
func NewDirectoryMsgClient(directoryThingID string, reqSink modules.IHiveModule) *DirectoryMsgClient {
	if directoryThingID == "" {
		directoryThingID = directory.DefaultDirectoryThingID
	}
	cl := &DirectoryMsgClient{
		directoryThingID: directoryThingID,
	}
	if reqSink != nil {
		cl.SetRequestSink(reqSink.HandleRequest)
		// notifications returned are passed to this client (if any subscriptions are made)
		reqSink.SetNotificationSink(cl.HandleNotification)
	}
	return cl
}

// Update a Thing TD in the directory and wait for confirmation
// This retuns nil if success or an error if something went wrong.
//
// NOTE this is intended for use by agents while the above DirectoryClient methods
// are intended for use by consumers. Since NewDirectoryMsgClient overwrites the
// notification sinks, any agent notification would be lost.
// Instead this method uses the given agent request handler to send the request.
//
// directoryServiceID is the thing ID of the directory service instance. Defaults to the module type
// tdJson is the TD in JSON to update in the directory.
// reqHandler is the request handler of the agent to send the request through.
func UpdateTD(directoryThingID string, tdJson string, reqHandler msg.RequestHandler) error {
	if directoryThingID == "" {
		directoryThingID = directory.DirectoryModuleType
	}
	req := msg.NewRequestMessage(
		td.OpInvokeAction, directoryThingID, directory.ActionUpdateThing, tdJson, "")
	_, err := msg.ForwardRequestWait(req, reqHandler)

	return err
}
