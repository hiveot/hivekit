package direct

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/messaging"
)

// This simple module is a simple SME passthrough that injects a clientID as a sender.
// Used for testing messaging between modules when no transport is used.
// This implements the IHiveModule interface
type DirectClientTransport struct {
	modules.HiveModuleBase
	clientID string
	sink     modules.IHiveModule
}

func (m *DirectClientTransport) HandleRequest(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
	req.SenderID = m.clientID
	return m.sink.HandleRequest(req)
}

// assign the clientID as the sender. This modifies the notification
func (m *DirectClientTransport) HandleNotification(notif *messaging.NotificationMessage) {
	notif.SenderID = m.clientID
	m.sink.HandleNotification(notif)
}

// Return a transport module that passes messages directly to a sink, using the given client as sender.
// Mainly intended for testing to inject the clientID.
func NewDirectTransport(clientID string, sink modules.IHiveModule) modules.IHiveModule {
	t := &DirectClientTransport{
		clientID: clientID,
		sink:     sink,
	}
	return t
}
