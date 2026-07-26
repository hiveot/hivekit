package transport

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api/msg"
	jsoniter "github.com/json-iterator/go"
)

// JSON encoder encoder for RRN messages.
// This implements the IMessageConverter interface.
type RRNJsonEncoder struct {
}

// DecodeNotification passes the notification message as-is
//
//	senderID is the server provided ID of the authenticated client.
//	raw is the json serialized encoded message
func (svc *RRNJsonEncoder) DecodeNotification(senderID string, raw []byte) (*msg.NotificationMessage, error) {

	var notif msg.NotificationMessage
	err := jsoniter.Unmarshal(raw, &notif)
	if err != nil {
		return nil, fmt.Errorf("DecodeNotification: Not a HiveOT RRN notification message: %w", err)
	}
	if senderID != "" {
		notif.SenderID = senderID
	}
	return &notif, nil
}

// DecodeRequest passes the request message as-is
//
//	senderID is the server provided ID of the authenticated client.
//	raw is the json serialized encoded message
func (svc *RRNJsonEncoder) DecodeRequest(senderID string, raw []byte) (*msg.RequestMessage, error) {

	var req msg.RequestMessage
	err := jsoniter.Unmarshal(raw, &req)
	if err != nil {
		return nil, fmt.Errorf("DecodeRequest: Not a HiveOT RRN Request message: %w", err)
	}
	if senderID != "" {
		req.SenderID = senderID
	}
	return &req, nil
}

// DecodeResponse passes the response message as-is
//
//	senderID is the server provided ID of the authenticated client.
//	raw is the json serialized encoded message
func (svc *RRNJsonEncoder) DecodeResponse(senderID string, raw []byte) (*msg.ResponseMessage, error) {

	var resp msg.ResponseMessage
	err := jsoniter.Unmarshal(raw, &resp)
	if err != nil {
		return nil, fmt.Errorf("DecodeResponse: Not a HiveOT RRN Response message: %w", err)
	}
	if senderID != "" {
		resp.SenderID = senderID
	}
	return &resp, nil
}

// determine the type of WSS message
func (svc *RRNJsonEncoder) DetermineMessageType(raw []byte) string {
	var rxType struct {
		MessageType string `json:"messageType"`
	}
	_ = jsoniter.Unmarshal(raw, &rxType)
	return rxType.MessageType
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
