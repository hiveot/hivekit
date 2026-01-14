package transports

import "github.com/hiveot/hivekit/go/msg"

// IMessageConverter converts between the RRN request-response-notification message
// envelopes and the underlying protocol specific message format.
//
// This is used by both server and client side to translate protocol messages
// to 'standard RRN'.
type IMessageConverter interface {
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

	// EncodeNotification converts a hiveot NotificationMessage to a native serialized protocol message
	// return an error if the message cannot be converted.
	EncodeNotification(notif *msg.NotificationMessage) ([]byte, error)

	// EncodeRequest converts a hiveot RequestMessage to a native serialized protocol message
	// return an error if the message cannot be converted.
	EncodeRequest(req *msg.RequestMessage) ([]byte, error)

	// EncodeResponse converts a hiveot ResponseMessage to a native serialized protocol message
	// This returns an error response if the message cannot be converted
	EncodeResponse(resp *msg.ResponseMessage) ([]byte, error)

	// GetProtocolType provides the protocol type for these messages,
	// eg ProtocolTypeWSS
	GetProtocolType() string
}
