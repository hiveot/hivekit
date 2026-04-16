package testenv

import (
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// TestTransport is a direct transport module to connect consumer and
// producer modules as if they were connected via a network transport,
// but without the overhead of setting up a transport server and client.
//
// Intended for testing the messaging between client and server side of a module.
//
// This implements the IHiveTransport interface
type TestTransport struct {
	transports.TransportServerBase

	// The senderID this transport represents (simulate a single connection)
	senderID string
}

// AddTDSecForms does nothing for a direct connection
func (srv *TestTransport) AddTDSecForms(tdi *td.TD, includeAffordances bool) {
}

// Receive a notification from the sink and pass it on to the notification sink (the consumer)
func (m *TestTransport) HandleNotification(notif *msg.NotificationMessage) {
	m.ForwardNotification(notif)
}

// Receive a request and forward it on to the sinks.
func (m *TestTransport) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	req.SenderID = m.senderID
	return m.ForwardRequest(req, replyTo)
}

// SendNotification sends a notification message to the consumer.
// This would mean that the client's remote side receives a notification.
// Since this doesn't do subscriptions, all notifications are received.
func (m *TestTransport) SendNotification(notif *msg.NotificationMessage) {
	m.ForwardNotification(notif)
}

// SendRequest sends a request message via the transport to the producer.
// In a direct transport this is the registered sink, pretending to be the remote server.
// Note this only has a single connection.
func (m *TestTransport) SendRequest(
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
func (m *TestTransport) SetAuthenticationHandler(h transports.ValidateTokenHandler) {
	_ = h
}

// assign the handler of new incoming connections
// func (m *DirectClientTransport) SetConnectionHandler(h transports.ConnectionHandler) {
// 	_ = h
// }

func (m *TestTransport) Start() (err error) {
	return nil
}

// Stop disconnects clients and remove connection listening
func (m *TestTransport) Stop() {
}

// NewTestTransport returns a transport module that passes messages from a consumer to a producer
// This sets the producer as the destination for requests and this module as
// the destination for producer notifications.
func NewTestTransport(
	senderID string, producer modules.IHiveModule) modules.IHiveModule {
	t := &TestTransport{
		senderID: senderID,
	}
	t.Init(senderID, "", "", "", nil)
	producer.SetNotificationSink(t.HandleNotification)
	t.SetRequestSink(producer.HandleRequest)
	var _ transports.ITransportServer = t // interface check
	var _ modules.IHiveModule = t         // interface check
	return t
}
