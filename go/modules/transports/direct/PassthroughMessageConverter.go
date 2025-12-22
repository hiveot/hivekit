package direct

import (
	"github.com/hiveot/hivekit/go/lib/messaging"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	jsoniter "github.com/json-iterator/go"
)

// Passthrough message converter simply passes RRN messages as-is.
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

// EncodeNotification passes the notification message as-is
func (svc *PassthroughMessageConverter) EncodeNotification(req *msg.NotificationMessage) (any, error) {
	// ensure this field is present as it is needed for decoding
	req.MessageType = messaging.MessageTypeNotification
	return req, nil
}

// EncodeRequest passes the request message as-is
func (svc *PassthroughMessageConverter) EncodeRequest(req *msg.RequestMessage) (any, error) {
	// ensure this field is present as it is needed for decoding
	req.MessageType = messaging.MessageTypeRequest
	return req, nil
}

// EncodeResponse passes the response message as-is
func (svc *PassthroughMessageConverter) EncodeResponse(resp *msg.ResponseMessage) any {
	// ensure this field is present as it is needed for decoding
	resp.MessageType = messaging.MessageTypeResponse
	return resp
}

// GetProtocolType returns the hiveot WSS protocol type identifier
func (svc *PassthroughMessageConverter) GetProtocolType() string {
	return transports.ProtocolTypeWSS
}

// Create a new instance of the hiveot passthrough message converter
func NewPassthroughMessageConverter() *PassthroughMessageConverter {
	return &PassthroughMessageConverter{}
}
