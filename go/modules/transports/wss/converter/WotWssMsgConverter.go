package converter

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	jsoniter "github.com/json-iterator/go"
)

// Messaging converter between WoT websocket and the HiveOT standard RRN message envelopes.

// Websocket notification message with all possible fields for all operations
type WotWssNotificationMessage struct {
	msg.NotificationMessage
	MessageType string `json:"messageType"` // notification
}

// Websocket requests message with all possible fields for all operations
type WotWssRequestMessage struct {
	msg.RequestMessage
	// queryaction
	ActionID string `json:"actionID,omitempty"` // input for operation
	// readmultipleproperties: array of property names
	Names []string `json:"names,omitempty"`
	// writeallproperties, writemultipleproperties input
	Values any `json:"values,omitempty"`
}
type WotWssActionStatus struct {
	ActionID      string          `json:"actionID"`
	Error         *msg.ErrorValue `json:"error,omitempty"`
	Output        any             `json:"output,omitempty"` // when completed
	State         string          `json:"state"`            // completed, failed, ...
	TimeRequested string          `json:"timeRequested"`
	TimeEnded     string          `json:"timeEnded,omitempty"` // when completed
}

// Websocket response message with all possible fields for all operations
type WotWssResponseMessage struct {
	CorrelationID string `json:"correlationID,omitempty"`

	// Error contains the short error description when status is failed.
	// Matches RFC9457 https://www.rfc-editor.org/rfc/rfc9457
	Error *msg.ErrorValue `json:"error"`

	// MessageID unique ID of the message. Intended to detect duplicates.
	MessageID string `json:"messageID,omitempty"`

	// This is set to the value of MessageTypeResponse
	MessageType string `json:"messageType"`

	// Name of the action or property affordance this is a response from.
	Name string `json:"name"`

	// The operation this is a response to. This MUST be the operation provided in the request.
	Operation string `json:"operation"`

	// invokeaction output value (synchronous)
	// for hiveot clients: readallproperties, readmultipleproperties ThingValue map
	Output any `json:"output,omitempty"`

	// invokeaction (async), queryaction response contains status
	Status *WotWssActionStatus `json:"status,omitempty"`

	// queryallactions response:
	Statuses map[string]WotWssActionStatus `json:"statuses,omitempty"`

	// ThingID of the thing this is a response from.
	ThingID string `json:"thingID,omitempty"`

	// readallproperties,readmultipleproperties,
	// writeallproperties, writemultipleproperties:
	// object with property name-value pairs
	Values any `json:"values,omitempty"`
}

// Websocket message converter converts requests, responses and notifications
// between hiveot standardized RRN envelope and the WoT websocket protocol (draft) messages.
// Websocket messages vary based on the operation.
// This implements the IMessageConverter interface
type WotWssMsgConverter struct {
}

// DecodeNotification converts a websocket notification to a hiveot notification message.
// Raw is the json serialized encoded message
func (svc *WotWssMsgConverter) DecodeNotification(raw []byte) *msg.NotificationMessage {

	var wssnotif WotWssNotificationMessage
	err := jsoniter.Unmarshal(raw, &wssnotif)
	//err := tputils.DecodeAsObject(msg, &notif)
	if err != nil || wssnotif.MessageType != msg.MessageTypeNotification {
		return nil
	}
	notifmsg := &wssnotif.NotificationMessage
	return notifmsg
}

// DecodeRequest converts a websocket request message to a hiveot request message.
// Raw is the json serialized encoded message.
// Websocket request messages are nearly identical to hiveot, so use passthrough.
// Conversion by operation:
// - cancelaction: copy wss actionID field to input
// - invokeaction: none
// - queryaction: copy wss actionID field to input
// - queryallactions: none
// - writeproperty:
func (svc *WotWssMsgConverter) DecodeRequest(raw []byte) *msg.RequestMessage {

	var wssreq WotWssRequestMessage
	err := jsoniter.Unmarshal(raw, &wssreq)

	//err := tputils.DecodeAsObject(msg, &req)
	if err != nil || wssreq.MessageType != msg.MessageTypeRequest {
		return nil
	}
	// query/cancel action messages carry an actionID in the request
	// (tentative: https://github.com/w3c/web-thing-protocol/issues/43)
	reqmsg := &wssreq.RequestMessage
	switch wssreq.Operation {

	case wot.OpQueryAction, wot.OpCancelAction:
		// input is actionID
		reqmsg.Input = wssreq.ActionID
	}
	return reqmsg
}

// DecodeResponse converts a websocket response message to a hiveot response message.
// Raw is the json serialized encoded message
func (svc *WotWssMsgConverter) DecodeResponse(raw []byte) *msg.ResponseMessage {

	var wssResp WotWssResponseMessage
	err := jsoniter.Unmarshal(raw, &wssResp)
	if err != nil {
		slog.Warn("DecodeResponse: Can't unmarshal websocket response", "error", err, "raw", string(raw))
		return nil
	}
	if wssResp.MessageType != msg.MessageTypeResponse {
		return nil
	}

	respMsg := msg.NewResponseMessage(
		wssResp.Operation, wssResp.ThingID, wssResp.Name,
		wssResp.Output,
		err,
		msg.StatusCompleted,
		wssResp.CorrelationID)

	// if the response is an error response then no need to decode any further
	if wssResp.Error != nil {
		return respMsg
	}

	switch wssResp.Operation {

	case wot.OpCancelAction:
		// hiveot response API doesnt contain the actionID. This is okay as the sender knows it.
	case wot.OpInvokeAction:
		// if wss contains a status object, use it
		if wssResp.Status != nil {
			respMsg.State = wssResp.Status.State
			// respMsg.Timestamp = wssResp.Status.TimeRequested
			respMsg.Timestamp = wssResp.Status.TimeEnded
		}

	case wot.OpQueryAction:
		// ResponseMessage contains a WSS ActionStatus object.
		// which is converted to a HiveOT ResponseMessage as the value.
		var wssStatus WotWssActionStatus
		err = utils.Decode(wssResp.Status, &wssStatus)
		if err != nil {
			return nil
		}
		// reconstruct the action response
		// Note that hiveOT uses correlationID instead of actionID
		// hiveot also doesn't return timerequested
		output := msg.ResponseMessage{
			Operation:     wot.OpInvokeAction,
			ThingID:       wssResp.ThingID,
			Name:          wssResp.Name,
			Output:        wssStatus.Output,
			State:         wssStatus.State,
			CorrelationID: wssResp.CorrelationID,
			Timestamp:     wssStatus.TimeEnded,
			Error:         wssResp.Error,
		}
		respMsg.Output = output

	case wot.OpQueryAllActions:
		// WSS ResponseMessage contains ActionStatus map
		var wssStatusMap map[string]WotWssActionStatus
		output := make(map[string]msg.ResponseMessage)
		err = utils.Decode(wssResp.Statuses, &wssStatusMap)
		if err != nil {
			return nil
		}
		// reconstruct the latest responses for the actions
		for name, wssStatus := range wssStatusMap {
			output[name] = msg.ResponseMessage{
				Operation:     wot.OpInvokeAction,
				ThingID:       wssResp.ThingID,
				Name:          name,
				Output:        wssStatus.Output,
				State:         wssStatus.State,
				CorrelationID: wssResp.CorrelationID,
				Timestamp:     wssStatus.TimeEnded,
				Error:         wssResp.Error,
			}
		}
		respMsg.Output = output

	case wot.OpReadAllProperties, wot.OpReadMultipleProperties,
		wot.OpWriteMultipleProperties:

		// the 'Value' property from the msg.ResponseMessage embedded struct
		// already contains the msg.ThingValue map.
		// Convert the websocket 'Values' field k-v map to ThingValue map
		tvMap := make(map[string]msg.ThingValue)
		if wssResp.Values != nil {
			wssPropValues := make(map[string]any)
			utils.DecodeAsObject(wssResp.Values, wssPropValues)
			for propName, propValue := range wssPropValues {
				tv := msg.ThingValue{
					AffordanceType: msg.AffordanceTypeProperty,
					Name:           propName,
					Data:           propValue,
					ThingID:        wssResp.ThingID,
					// Timestamp: n/a
				}
				tvMap[tv.Name] = tv
			}
			respMsg.Output = tvMap
		}
	}
	return respMsg
}

// EncodeNotification converts a hiveot RequestMessage to a websocket equivalent message
func (svc *WotWssMsgConverter) EncodeNotification(notif *msg.NotificationMessage) ([]byte, error) {
	wssNotif := WotWssNotificationMessage{
		NotificationMessage: *notif,
	}
	// ensure this field is present as it is needed for decoding
	wssNotif.MessageType = msg.MessageTypeNotification
	return jsoniter.Marshal(wssNotif)
}

// EncodeRequest converts a hiveot RequestMessage to websocket equivalent message
func (svc *WotWssMsgConverter) EncodeRequest(req *msg.RequestMessage) ([]byte, error) {
	wssReq := WotWssRequestMessage{
		RequestMessage: *req,
		ActionID:       req.CorrelationID,
	}
	// ensure this field is present as it is needed for decoding
	wssReq.MessageType = msg.MessageTypeRequest
	switch req.Operation {
	case wot.OpWriteMultipleProperties:
		wssReq.Values = req.Input
	case wot.OpQueryAction:
		// correlationID is used as actionID
		wssReq.ActionID = req.CorrelationID
	}
	return jsoniter.Marshal(wssReq)
}

// EncodeResponse converts a hiveot ResponseMessage to websocket equivalent message
// This always returns a response
func (svc *WotWssMsgConverter) EncodeResponse(resp *msg.ResponseMessage) ([]byte, error) {
	var err error

	// the hiveot response message partially identical to the websocket message
	// only convert where they differ
	wssResp := WotWssResponseMessage{
		CorrelationID: resp.CorrelationID,
		// the hiveot error response is identical to the websocket error response
		Error:       resp.Error,
		MessageID:   resp.MessageID,
		MessageType: resp.MessageType,
		Name:        resp.Name,
		Operation:   resp.Operation,
		Output:      resp.Output,
		Status:      nil,
		Statuses:    nil,
		ThingID:     resp.ThingID,
		Values:      nil,
	}
	// last, set status(es) and values, depending on the operation
	switch resp.Operation {
	case wot.OpCancelAction:
		// actionID of cancelled action ?
		// wssResp.ActionID = resp.CorrelationID
	case wot.OpInvokeAction:
		wssResp.Status = &WotWssActionStatus{
			// websocket asynchronous response returns ActionID
			ActionID:  resp.CorrelationID,
			Error:     resp.Error, // error fields are identical
			State:     resp.State,
			Output:    resp.Output,
			TimeEnded: resp.Timestamp,
		}
	case wot.OpQueryAction:
		// the output is the last response: convert it to an ActionStatus object
		qResp := msg.ResponseMessage{}
		utils.Decode(resp.Output, &qResp)
		wssResp.Status = &WotWssActionStatus{
			ActionID:  qResp.CorrelationID,
			Error:     qResp.Error, // error fields are identical
			Output:    qResp.Output,
			State:     qResp.State,
			TimeEnded: qResp.Timestamp,
		}
	case wot.OpQueryAllActions:
		// convert action responses in output to a  WotWssActionStatuses map
		var hiveotActionStatusMap map[string]msg.ResponseMessage
		err = utils.Decode(resp.Output, &hiveotActionStatusMap)
		if err != nil {
			err = fmt.Errorf("Can't convert ResponseMessage map response to websocket actionstatus type. "+
				"Response does not contain ResponseMessage map. "+
				"thingID='%s'; name='%s'; operation='%s'; Received '%s'; Error='%s'",
				resp.ThingID, resp.Name, resp.Operation,
				utils.DecodeAsString(resp.Output, 200), err.Error())
			wssResp.Error = msg.ErrorValueFromError(err)
		}
		wssStatusMap := make(map[string]WotWssActionStatus)
		for _, respMsg := range hiveotActionStatusMap {
			wssStatusMap[respMsg.Name] = WotWssActionStatus{
				ActionID:  respMsg.CorrelationID,
				Error:     respMsg.Error, // error fields are identical
				State:     respMsg.State,
				TimeEnded: respMsg.Timestamp,
				Output:    respMsg.Output,
			}
		}
		wssResp.Statuses = wssStatusMap
	case wot.OpReadAllProperties, wot.OpReadMultipleProperties:
		// convert ThingValue map to map of name-value pairs
		// the last updated timestamp is lost.
		var thingValueList map[string]msg.ThingValue
		err = utils.DecodeAsObject(resp.Output, &thingValueList)
		if err != nil {
			err = fmt.Errorf("encodeResponse (%s). Not a ThingValue map; err: %w", resp.Operation, err)
			wssResp.Error = msg.ErrorValueFromError(err)
		}
		wssPropValues := make(map[string]any)
		for _, thingValue := range thingValueList {
			wssPropValues[thingValue.Name] = thingValue.ToString(0)
		}
		wssResp.Values = wssPropValues
		// Note that wssResp also includes the ResponseMessage 'Value' property
		// which hiveot clients can use to obtain the ThingValue result.
		// non-hiveot clients will see the key-value map in 'Values'
	case wot.OpReadProperty:
	}

	return jsoniter.Marshal(wssResp)
}

// GetProtocolType returns the hiveot WSS protocol type identifier
func (svc *WotWssMsgConverter) GetProtocolType() string {
	return transports.ProtocolTypeWotWSS
}

// Create a new instance of the WoT websocket to hiveot message converter
func NewWotWssMsgConverter() *WotWssMsgConverter {
	return &WotWssMsgConverter{}
}
