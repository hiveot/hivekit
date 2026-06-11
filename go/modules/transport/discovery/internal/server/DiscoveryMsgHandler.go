package internal

import (
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/directory"
)

// When a request to create a TD is received then serve it in discovery
// Intended for use in a module chain where the device publishes its TD.
func (m *DiscoveryServer) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	// intercept a directory update to publish a TD
	if req.Operation == td.OpInvokeAction && req.ThingID == m.directoryThingID &&
		(req.Name == directory.CreateThingAction || req.Name == directory.UpdateThingAction) {

		tdJson := req.ToString(0)
		m.ServeThingTD(tdJson)

		// forward it. ignore error if this is the last step in the chain
		_ = m.HiveModuleBase.HandleRequest(req, replyTo)
		return nil
	} else {
		return m.HiveModuleBase.HandleRequest(req, replyTo)
	}
}
