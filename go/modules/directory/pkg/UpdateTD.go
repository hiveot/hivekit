package directorypkg

import (
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/directory"
)

// Update a Thing TD on the directory and wait for confirmation.
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
		directoryThingID = directory.DirectoryServiceModuleType
	}
	req := msg.NewRequestMessage(
		td.OpInvokeAction, directoryThingID, directory.UpdateThingAction, tdJson)

	_, err := msg.ForwardRequestWait(req, reqHandler, msg.DefaultRnRTimeout)

	return err
}
