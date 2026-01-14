package direct

import (
	"github.com/hiveot/hivekit/go/lib/messaging"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	jsoniter "github.com/json-iterator/go"
)

// Passthrough message converter simply passes serialized RRN messages.
// This implements the IMessageConverter interface.
type PassthroughMessageConverter struct {
}

// DecodeNotification passes the notification message as-is
// Raw is the json serialized encoded message
func (svc *PassthroughMessageConverter) DecodeNotification(raw []byte) *msg.NotificationMessage {

	var notif msg.NotificationMessage
	err := jsoniter.Unmarshal(raw, &notif)
	//err := tputils.DecodeAsObject(msg, &notif)
	if err != nil || notif.MessageType != messaging.MessageTypeNotification {
		return nil
	}
	return &notif
}

// DecodeRequest passes the request message as-is
// Raw is the json serialized encoded message
func (svc *PassthroughMessageConverter) DecodeRequest(raw []byte) *msg.RequestMessage {

	var req msg.RequestMessage
	err := jsoniter.Unmarshal(raw, &req)
	//err := tputils.DecodeAsObject(msg, &req)
	if err != nil || req.MessageType != messaging.MessageTypeRequest {
		return nil
	}
	return &req
}

// DecodeResponse passes the response message as-is
// Raw is the json serialized encoded message
func (svc *PassthroughMessageConverter) DecodeResponse(
	raw []byte) *msg.ResponseMessage {

	var resp msg.ResponseMessage
	err := jsoniter.Unmarshal(raw, &resp)
	if err != nil || resp.MessageType != messaging.MessageTypeResponse {
		return nil
	}
	return &resp
}

// EncodeNotification serializes the notification message as-is
func (svc *PassthroughMessageConverter) EncodeNotification(
	notif *msg.NotificationMessage) ([]byte, error) {

	// ensure this field is present as it is needed for decoding
	notif.MessageType = messaging.MessageTypeNotification
	return jsoniter.Marshal(notif)
}

// EncodeRequest serializes the request message as-is
func (svc *PassthroughMessageConverter) EncodeRequest(req *msg.RequestMessage) ([]byte, error) {
	// ensure this field is present as it is needed for decoding
	req.MessageType = messaging.MessageTypeRequest
	return jsoniter.Marshal(req)
}

// EncodeResponse serializes the response message as-is
func (svc *PassthroughMessageConverter) EncodeResponse(resp *msg.ResponseMessage) ([]byte, error) {
	// ensure this field is present as it is needed for decoding
	resp.MessageType = messaging.MessageTypeResponse
	return jsoniter.Marshal(resp)
}

// GetProtocolType returns the hiveot WSS protocol type identifier
func (svc *PassthroughMessageConverter) GetProtocolType() string {
	return transports.ProtocolTypeWotWSS
}

// Create a new instance of the hiveot passthrough message converter
func NewPassthroughMessageConverter() *PassthroughMessageConverter {
	return &PassthroughMessageConverter{}
}
