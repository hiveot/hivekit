package transports

import "github.com/hiveot/hivekit/go/modules/messaging"

// IMessageConverter converts between the standardized message envelope (SME)
// and the underlying protocol specific message format.
//
// This is used to convert between te SME and transport protocols,
// including the WoT websocket protocol, HttpBasic/SSE-SC protocol,
// MQTT protocol.
//
// Intended for use by consumers and agents on the client and server side.
type IMessageConverter interface {
	// DecodeNotification converts a protocol message to a hiveot notification message
	// provide the serialized data to avoid multiple unmarshalls
	// This returns nil if this isn't a notification.
	DecodeNotification(raw []byte) *messaging.NotificationMessage

	// DecodeRequest converts a protocol message to a hiveot request message
	// provide the serialized data to avoid multiple unmarshalls
	// This returns nil if this isn't a request.
	DecodeRequest(raw []byte) *messaging.RequestMessage

	// DecodeResponse converts a protocol message to a hiveot response message.
	// This returns nil if this isn't a response
	DecodeResponse(raw []byte) *messaging.ResponseMessage

	// EncodeNotification converts a hiveot NotificationMessage to a native protocol message
	// return an error if the message cannot be converted.
	EncodeNotification(notif *messaging.NotificationMessage) (any, error)

	// EncodeRequest converts a hiveot RequestMessage to a native protocol message
	// return an error if the message cannot be converted.
	EncodeRequest(req *messaging.RequestMessage) (any, error)

	// EncodeResponse converts a hiveot ResponseMessage to a native protocol message
	// This returns an error response if the message cannot be converted
	EncodeResponse(resp *messaging.ResponseMessage) any

	// GetProtocolType provides the protocol type for these messages,
	// eg ProtocolTypeWSS
	GetProtocolType() string
}
