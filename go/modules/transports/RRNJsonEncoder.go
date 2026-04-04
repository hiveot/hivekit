package transports

import (
	"fmt"

	"github.com/hiveot/hivekit/go/msg"
	jsoniter "github.com/json-iterator/go"
)

// JSON encoder encoder for RRN messages.
// This implements the IMessageConverter interface.
type RRNJsonEncoder struct {
}

// DecodeNotification passes the notification message as-is
// Raw is the json serialized encoded message
func (svc *RRNJsonEncoder) DecodeNotification(raw []byte) (*msg.NotificationMessage, error) {

	var notif msg.NotificationMessage
	err := jsoniter.Unmarshal(raw, &notif)
	if err != nil {
		return nil, fmt.Errorf("DecodeNotification: unmarshal error: %w", err)
	}

	if notif.AffordanceType == "" {
		return nil, fmt.Errorf("DecodeRequest: Message is not a NotificationMessage")
	}
	return &notif, nil
}

// DecodeRequest passes the request message as-is
// Raw is the json serialized encoded message
func (svc *RRNJsonEncoder) DecodeRequest(raw []byte) (*msg.RequestMessage, error) {

	var req msg.RequestMessage
	err := jsoniter.Unmarshal(raw, &req)
	if err != nil {
		return nil, fmt.Errorf("DecodeRequest: unmarshal error: %w", err)
	}
	if req.MessageType != msg.MessageTypeRequest {
		return nil, fmt.Errorf("DecodeRequest: Message is not a RequestMessage")
	}
	return &req, nil
}

// DecodeResponse passes the response message as-is
// Raw is the json serialized encoded message
func (svc *RRNJsonEncoder) DecodeResponse(raw []byte) (*msg.ResponseMessage, error) {

	var resp msg.ResponseMessage
	err := jsoniter.Unmarshal(raw, &resp)
	if err != nil {
		return nil, fmt.Errorf("DecodeResponse: unmarshal error: %w", err)
	}
	if resp.MessageType != msg.MessageTypeResponse {
		return nil, fmt.Errorf("Message isn't a ResponseMessage")
	}
	return &resp, nil
}

// EncodeNotification serializes the notification message as-is
func (svc *RRNJsonEncoder) EncodeNotification(
	notif *msg.NotificationMessage) ([]byte, error) {
	// ensure this field is present as it is needed for decoding
	notif.MessageType = msg.MessageTypeNotification
	return jsoniter.Marshal(notif)
}

// EncodeRequest serializes the request message as-is
func (svc *RRNJsonEncoder) EncodeRequest(req *msg.RequestMessage) ([]byte, error) {
	// ensure this field is present as it is needed for decoding
	req.MessageType = msg.MessageTypeRequest
	return jsoniter.Marshal(req)
}

// EncodeResponse serializes the response message as-is
func (svc *RRNJsonEncoder) EncodeResponse(resp *msg.ResponseMessage) ([]byte, error) {
	// ensure this field is present as it is needed for decoding
	resp.MessageType = msg.MessageTypeResponse
	return jsoniter.Marshal(resp)
}

// GetProtocolType returns the hiveot  protocol type identifier
// func (svc *PassthroughMessageConverter) GetProtocolType() string {
// 	return td.PassthroughProtocolType
// }

// Create a new instance of the hiveot RRN message encoder
func NewRRNJsonEncoder() *RRNJsonEncoder {
	return &RRNJsonEncoder{}
}
