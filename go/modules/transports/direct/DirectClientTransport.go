package direct

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot/td"
)

// DirectClientTransport is a simple RRN passthrough that injects a clientID
// as a sender. Intended to simulate a client connection to a module without
// all the steps of setting up a protocol server and connecting as a client.
//
// Used for testing messaging between modules when no transport is used.
// This implements the IHiveModule interface
type DirectClientTransport struct {
	transports.TransportServerBase
}

// AddTDForms does nothing for a direct connection
func (srv *DirectClientTransport) AddTDForms(tdi *td.TD, includeAffordances bool) {
}

// Receive a notification from the sink and pass it on to the notification sink (the consumer)
func (m *DirectClientTransport) HandleNotification(notif *msg.NotificationMessage) {
	m.ForwardNotification(notif)
}

// Receive a request and forward it on to the sinks.
func (m *DirectClientTransport) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	req.SenderID = m.GetModuleID()
	return m.ForwardRequest(req, replyTo)
}

// SendNotification sends a notification message to the consumer.
// This would mean that the client's remote side receives a notification.
// Since this doesn't do subscriptions, all notifications are received.
func (m *DirectClientTransport) SendNotification(notif *msg.NotificationMessage) {
	m.ForwardNotification(notif)
}

// SendRequest sends a request message via the transport to the producer.
// In a direct transport this is the registered sink, pretending to be the remote server.
// Note this only has a single connection.
func (m *DirectClientTransport) SendRequest(
	clientID string, req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	err = m.ForwardRequest(req, replyTo)
	return err
}

// SendResponse sends a response message to the consumer,
// // If the consumer is not connected this returns an error, otherwise nil.
// func (m *DirectClientTransport) SendResponse(
// 	clientID, cid string, resp *msg.ResponseMessage) (err error) {

// 	if m.producer != nil {
// 		// err = m.source.onResponse(resp)
// 	}
// 	return err
// }

// assign the authenticator of incoming connections
func (m *DirectClientTransport) SetAuthenticationHandler(h transports.AuthenticationHandler) {
	_ = h
}

// assign the handler of new incoming connections
// func (m *DirectClientTransport) SetConnectionHandler(h transports.ConnectionHandler) {
// 	_ = h
// }

func (m *DirectClientTransport) Start(yamlConfig string) (err error) {
	return nil
}

// Stop disconnects clients and remove connection listening
func (m *DirectClientTransport) Stop() {
}

// Return a transport module that passes messages from a consumer to a producer
// This sets the producer as the destination for requests and this module as
// the destination for producer notifications.
func NewDirectTransport(
	moduleID string, producer modules.IHiveModule) modules.IHiveModule {
	t := &DirectClientTransport{}
	t.Init(moduleID, "")
	producer.SetNotificationSink(t.HandleNotification)
	t.SetRequestSink(producer.HandleRequest)
	var _ transports.ITransportServer = t // interface check
	var _ modules.IHiveModule = t         // interface check
	return t
}
