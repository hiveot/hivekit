package directory_service

import (
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/directory"
)

// Update a Thing TD on the directory and wait for confirmation.
// This retuns nil if success or an error if something went wrong.
//
// NOTE this is intended for use by devices, while the DirectoryClient methods are intended
// for use by consumers.
//
//	directoryServiceID is the thing ID of the directory service instance. Defaults to DirectoryServiceCellType.
//	tdJson is the TD in JSON to update in the directory.
//	reqHandler is the request handler of the connection to send the request through.
func UpdateTD(directoryThingID string, tdJson string, reqHandler msg.RequestHandler) error {
	if directoryThingID == "" {
		directoryThingID = directory.DirectoryServiceCellType
	}
	req := msg.NewRequestMessage(
		td.OpInvokeAction, directoryThingID, directory.UpdateThingAction, tdJson)

	_, err := msg.EmitRequestWait(req, reqHandler, msg.DefaultRnRTimeout)

	return err
}
