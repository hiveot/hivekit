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
	transports.TransportModuleBase
	source modules.IHiveModule
}

// AddTDForms does nothing for a direct connection
func (srv *DirectClientTransport) AddTDForms(tdi *td.TD, includeAffordances bool) {
}

// Receive a notification and pass it on to the sinks.
// This sets the notification SenderID to the module ID.
func (m *DirectClientTransport) HandleNotification(notif *msg.NotificationMessage) {
	notif.SenderID = m.GetModuleID()
	m.ForwardNotification(notif)
}

// Receive a request and forward it on to the sinks.
// This is a module input.
func (m *DirectClientTransport) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	req.SenderID = m.GetModuleID()
	return m.ForwardRequest(req, replyTo)
}

// HandleResponse.
// client sends a response to the server as would happen in connection reversal.
func (m *DirectClientTransport) HandleResponse(resp *msg.ResponseMessage) error {
	moduleID := m.GetModuleID()
	sink := m.GetSink()
	if resp.ThingID == moduleID {
		// response is for this module. A subclass should implement this instead.
		// nothing to do here, please move along.
	}
	resp.SenderID = moduleID
	return sink.HandleResponse(resp)
}

// Sendrequest sends a request message to the source, eg client end
// This would mean that the client's remote side receives a notification.
// Since this doesn't do subscriptions, all notifications are received.
func (m *DirectClientTransport) SendNotification(notif *msg.NotificationMessage) {
	if m.source != nil {
		//m.source.onNotification(notif)
	}
}

// SendRequest sends a request message via the transport to the agent.
// This doesn't apply in a direct transport
// Note this only has a single connection.
func (m *DirectClientTransport) SendRequest(
	clientID string, req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	if m.source != nil {
		// err = m.source.onRequest(req, nil)
	}
	return err
}

// SendResponse sends a response message via the connection made by a consumer,
// identified by the connectionID (cid).
// If the consumer is not connected this returns an error, otherwise nil.
func (m *DirectClientTransport) SendResponse(
	clientID, cid string, resp *msg.ResponseMessage) (err error) {

	if m.source != nil {
		// err = m.source.onResponse(resp)
	}
	return err
}

// assign the authenticator of incoming connections
func (m *DirectClientTransport) SetAuthenticationHandler(h transports.AuthenticationHandler) {
	_ = h
}

// assign the handler of new incoming connections
func (m *DirectClientTransport) SetConnectionHandler(h transports.ConnectionHandler) {
	_ = h
}

// Return a transport module that passes messages from a source to a sink
func NewDirectTransport(
	moduleID string, source modules.IHiveModule, sink modules.IHiveModule) modules.IHiveModule {
	t := &DirectClientTransport{
		source: source,
	}
	t.Init(moduleID, sink, "")
	var _ transports.ITransportModule = t // interface check
	return t
}
