package testenv

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/transport"
)

// TestTransport is a direct transport client/server to connect consumer and
// producer cells as if they were connected via a network client and server transport,
// but without the overhead of setting up a transport server and client.
//
// The thingID of this transport is used as the senderID of requests. This
// simulates server cells that set the clientID of the connection as the sender.
//
// Intended for testing the messaging between client and server side of a cell.
//
// This implements the IHiveTransport interface
type TestTransport struct {
	*transport.TransportServerBase
}

// AddTDSecForms does nothing for a direct connection
func (srv *TestTransport) AddTDSecForms(tdi *td.TD, includeAffordances bool) {
}

// GetTD returns the server TD, containing connection and authentication information
func (srv *TestTransport) GetTD() *td.TD {
	return nil
}

// Receive a notification from the sink and sends it to the client.
func (srv *TestTransport) HandleNotification(notif *msg.NotificationMessage) {
	srv.SendNotification(notif)
}

// Receive a request and forward it on to the sinks.
func (srv *TestTransport) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	req.SenderID = srv.GetThingID()
	return srv.ForwardRequest(req, replyTo)
}

// SendNotification sends a notification message to the consumer.
// This would mean that the client's remote side receives a notification.
// Since this doesn't do subscriptions, all notifications are received.
func (srv *TestTransport) SendNotification(notif *msg.NotificationMessage) {
	srv.EmitNotification(notif)
}

// SendRequest sends a request message via the transport to the producer.
// In a direct transport this is the registered sink, pretending to be the remote server.
// Note this only has a single connection.
func (srv *TestTransport) SendRequest(
	clientID string, req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	err = srv.EmitRequest(req, replyTo)
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
func (srv *TestTransport) SetAuthenticationHandler(h api.ValidateTokenHandler) {
	_ = h
}

// NewTestTransport returns a transport cell that passes messages from a consumer to a producer
// This sets the producer as the request sink and this cell as the notification sink.
func NewTestTransport(
	thingID string, producer api.IHiveCell) api.IHiveCell {
	t := &TestTransport{
		TransportServerBase: transport.NewTransportServerBase(thingID, "", nil),
	}
	producer.SetNotificationSink(t)
	t.SetRequestSink(producer)
	var _ api.ITransportServer = t // interface check
	var _ api.IHiveCell = t        // interface check
	return t
}
