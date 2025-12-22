package transports

import "github.com/hiveot/hivekit/go/msg"

// IMsgConverter converts between the RRN request-response-notification envelope
// and the underlying protocol specific message format.
//
// This is used to convert between te RRN and transport protocols,
// including the WoT websocket protocol, HttpBasic/SSE-SC protocol,
// MQTT protocol.
//
// Intended for use by consumers and agents on the client and server side.
type IMsgConverter interface {
	// DecodeNotification converts a protocol message to a hiveot notification message
	// provide the serialized data to avoid multiple unmarshalls
	// This returns nil if this isn't a notification.
	DecodeNotification(raw []byte) *msg.NotificationMessage

	// DecodeRequest converts a protocol message to a hiveot request message
	// provide the serialized data to avoid multiple unmarshalls
	// This returns nil if this isn't a request.
	DecodeRequest(raw []byte) *msg.RequestMessage

	// DecodeResponse converts a protocol message to a hiveot response message.
	// This returns nil if this isn't a response
	DecodeResponse(raw []byte) *msg.ResponseMessage

	// EncodeNotification converts a hiveot NotificationMessage to a native protocol message
	// return an error if the message cannot be converted.
	EncodeNotification(notif *msg.NotificationMessage) (any, error)

	// EncodeRequest converts a hiveot RequestMessage to a native protocol message
	// return an error if the message cannot be converted.
	EncodeRequest(req *msg.RequestMessage) (any, error)

	// EncodeResponse converts a hiveot ResponseMessage to a native protocol message
	// This returns an error response if the message cannot be converted
	EncodeResponse(resp *msg.ResponseMessage) any

	// GetProtocolType provides the protocol type for these messages,
	// eg ProtocolTypeWSS
	GetProtocolType() string
}
