package transport

import "github.com/hiveot/hivekit/go/api/msg"

// IMessageEncoder encodes and decodes the RRN request-response-notification message
// envelopes into the underlying protocol specific message format.
//
// This is used by both server and client side to translate protocol messages
// to 'standard RRN'.
type IMessageEncoder interface {
	// Determine which message type is contained
	// This returns an empty name if the message is invalid
	DetermineMessageType(raw []byte) string

	// DecodeNotification converts a protocol message to a hiveot notification message
	// provide the serialized data to avoid multiple unmarshalls
	//
	// senderID sets the notification sender if known. Use "" client side.
	// This returns an error if this isn't a notification.
	DecodeNotification(senderID string, raw []byte) (*msg.NotificationMessage, error)

	// DecodeRequest converts a protocol message to a hiveot request message
	// provide the serialized data to avoid multiple unmarshalls
	//
	// senderID sets the request sender if known. Use "" client side.
	// This returns an error if this isn't a request.
	DecodeRequest(senderID string, raw []byte) (*msg.RequestMessage, error)

	// DecodeResponse converts a protocol message to a hiveot response message.
	// senderID sets the response sender if known. Use "" client side.
	// This returns an error if this isn't a response
	DecodeResponse(senderID string, raw []byte) (*msg.ResponseMessage, error)

	// EncodeNotification converts a hiveot NotificationMessage to a native serialized protocol message
	// return an error if the message cannot be converted.
	EncodeNotification(notif *msg.NotificationMessage) ([]byte, error)

	// EncodeRequest converts a hiveot RequestMessage to a native serialized protocol message
	// return an error if the message cannot be converted.
	EncodeRequest(req *msg.RequestMessage) ([]byte, error)

	// EncodeResponse converts a hiveot ResponseMessage to a native serialized protocol message
	// This returns an error response if the message cannot be converted
	EncodeResponse(resp *msg.ResponseMessage) ([]byte, error)
}
