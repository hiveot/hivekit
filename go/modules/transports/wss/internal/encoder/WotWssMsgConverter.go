package wssencoder

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/utils"
	jsoniter "github.com/json-iterator/go"
)

// operations related to event affordances
var EventAffOps = []string{
	td.OpSubscribeEvent, td.OpSubscribeAllEvents,
}

// operations related to action affordances
var ActionAffOps = []string{
	td.OpCancelAction, td.OpInvokeAction, td.OpQueryAction, td.OpQueryAllActions,
}

// operations related to action affordances
var PropAffOps = []string{
	td.OpObserveAllProperties, td.OpObserveMultipleProperties, td.OpObserveProperty,
	td.OpReadProperty, td.OpReadAllProperties, td.OpReadMultipleProperties,
	td.OpWriteProperty,
}

// Messaging converter between WoT websocket and the HiveOT standard RRN message envelopes.
type WotWssCommon struct {
	CorrelationID string `json:"correlationID,omitempty"`

	MessageID string `json:"messageID"`

	// one of "request", "response", "notification"
	MessageType string `json:"messageType"`

	// Name of the action or property affordance this is a response from.
	// Op: readproperty, writeproperty, observeproperty, unobserveproperty
	// Op: invokeaction
	// Op: subscribeevent, unsubscribeevent
	Name string `json:"name"`

	// the operation this is a request for or response to.
	Operation string `json:"operation"`

	// ThingID of the thing this is a response from.
	ThingID string `json:"thingID,omitempty"`

	// Time in date-time format [RFC3339] the message was created or notification occurred.
	Timestamp string `json:"timestamp"`
}

// Websocket notification message with all possible fields for all operations
type WotWssNotificationMessage struct {
	WotWssCommon

	// Data with the event payload
	// op: subscribeevent, subscribeallevents
	Data any `json:"data,omitempty"` // native

	// op: observeproperty, observeallproperties
	Value any `json:"value,omitempty"` // native
}

// Websocket requests message with all possible fields for all operations
type WotWssRequestMessage struct {
	WotWssCommon

	// op: queryaction, cancelaction
	// ActionID string `json:"actionID,omitempty"` // input for operation

	// op: invokeaction
	Input any `json:"input,omitempty"` // native

	//LastNotificationID string `json:"lastNotificationID"`

	// op; readmultipleproperties (array of property names)y
	Names []string `json:"names,omitempty"`

	// SenderID is an optional non-WoT field. Primarily intended for testing agents know who
	// send the request if they have a reverse connection.
	SenderID string `json:"senderID"`

	// op: writeproperty
	Value any `json:"value,omitempty"`

	// op: writeallproperties, writemultipleproperties input
	// map of {key:value} pairs
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
	WotWssCommon

	// Op: cancelaction
	// ActionID string `json:"actionID,omitempty"`

	// Error contains the short error description when status is failed.
	// Matches RFC9457 https://www.rfc-editor.org/rfc/rfc9457
	Error *msg.ErrorValue `json:"error"`

	// Op: invokeaction (synchronous action)
	// for hiveot clients: readallproperties, readmultipleproperties ThingValue map
	Output any `json:"output,omitempty"`

	// Op: invokeaction (async action), queryaction
	Status *WotWssActionStatus `json:"status,omitempty"`

	// Op: queryallactions response:
	Statuses map[string]WotWssActionStatus `json:"statuses,omitempty"`

	// Op: readproperty, writeproperty
	Value any `json:"value,omitempty"`

	// Op: readallproperties,readmultipleproperties,
	// Op: writeallproperties, writemultipleproperties:
	//
	// object with property name-value pairs
	Values any `json:"values,omitempty"`
}

// Websocket message converter converts requests, responses and notifications
// between hiveot standardized RRN envelope and the WoT websocket protocol (draft) messages.
// Websocket messages vary based on the operation.
// This implements the IMessageConverter interface
type WotWssMsgEncoder struct {
}

// DecodeNotification converts a websocket notification to a hiveot notification message.
// Raw is the json serialized encoded message
func (svc *WotWssMsgEncoder) DecodeNotification(raw []byte) (*msg.NotificationMessage, error) {

	var wssnotif WotWssNotificationMessage

	err := jsoniter.Unmarshal(raw, &wssnotif)
	if err != nil {
		return nil, fmt.Errorf("DecodeNotification: unmarshal error: %w", err)
	}
	if wssnotif.MessageType != msg.MessageTypeNotification {
		return nil, fmt.Errorf("DecodeRequest: Message is not a NotificationMessage")
	}

	affType := msg.AffordanceTypeProperty
	if slices.Contains(EventAffOps, wssnotif.Operation) {
		affType = msg.AffordanceTypeEvent
	} else if slices.Contains(ActionAffOps, wssnotif.Operation) {
		affType = msg.AffordanceTypeAction
	}
	notifmsg := &msg.NotificationMessage{
		AffordanceType: affType,
		CorrelationID:  wssnotif.CorrelationID,
		Data:           wssnotif.Data,
		MessageID:      wssnotif.MessageID,
		MessageType:    wssnotif.MessageType,
		Name:           wssnotif.Name,
		// Operation:      wssnotif.Operation,
		// SenderID:      wssreq.SenderID,
		ThingID:   wssnotif.ThingID,
		Timestamp: wssnotif.Timestamp,
	}
	return notifmsg, nil
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
func (svc *WotWssMsgEncoder) DecodeRequest(raw []byte) (*msg.RequestMessage, error) {

	var wssreq WotWssRequestMessage
	err := jsoniter.Unmarshal(raw, &wssreq)

	if err != nil {
		return nil, fmt.Errorf("DecodeRequest: unmarshal error: %w", err)
	}
	if wssreq.MessageType != msg.MessageTypeRequest {
		return nil, fmt.Errorf("DecodeRequest: Message is not a RequestMessage")
	}
	// query/cancel action messages carry an actionID in the request
	// (tentative: https://github.com/w3c/web-thing-protocol/issues/43)
	reqMsg := &msg.RequestMessage{
		CorrelationID: wssreq.CorrelationID,
		Input:         wssreq.Input,
		MessageID:     wssreq.MessageID,
		MessageType:   wssreq.MessageType,
		Name:          wssreq.Name,
		Operation:     wssreq.Operation,
		SenderID:      wssreq.SenderID,
		ThingID:       wssreq.ThingID,
		Timestamp:     wssreq.Timestamp,
	}
	switch wssreq.Operation {

	case td.OpQueryAction, td.OpCancelAction:
		// correlationID identifies the action to query/cancel
		// input is actionID
		// reqMsg.Input = wssreq.ActionID
	}
	return reqMsg, nil
}

// DecodeResponse converts a websocket response message to a hiveot response message.
// Raw is the json serialized encoded message
func (svc *WotWssMsgEncoder) DecodeResponse(raw []byte) (*msg.ResponseMessage, error) {

	var wssResp WotWssResponseMessage
	err := jsoniter.Unmarshal(raw, &wssResp)
	if err != nil {
		return nil, fmt.Errorf("DecodeResponse: unmarshal error: %w", err)
	}
	if wssResp.MessageType != msg.MessageTypeResponse {
		return nil, fmt.Errorf("Message isn't a ResponseMessage")
	}

	respMsg := &msg.ResponseMessage{
		CorrelationID: wssResp.CorrelationID,
		// Input:         wssResp.Input,
		MessageID:   wssResp.MessageID,
		MessageType: wssResp.MessageType,
		Name:        wssResp.Name,
		Operation:   wssResp.Operation,
		Output:      wssResp.Output,
		// SenderID:      wssreq.SenderID,
		Status:    msg.StatusCompleted,
		ThingID:   wssResp.ThingID,
		Timestamp: wssResp.Timestamp,
	}

	// if the response is an error response then no need to decode any further
	if wssResp.Error != nil {
		return respMsg, nil
	}

	switch wssResp.Operation {

	case td.OpCancelAction:
		// hiveot response API doesnt contain the actionID. This is okay as the sender knows it.
	case td.OpInvokeAction:
		// if wss contains a status object, use it
		if wssResp.Status != nil {
			respMsg.Status = wssResp.Status.State
			// respMsg.Timestamp = wssResp.Status.TimeRequested
			respMsg.Timestamp = wssResp.Status.TimeEnded
		}

	case td.OpQueryAction:
		// ResponseMessage contains a WSS ActionStatus object.
		// which is converted to a HiveOT ResponseMessage as the value.
		var wssStatus WotWssActionStatus
		err = utils.Decode(wssResp.Status, &wssStatus)
		if err != nil {
			return nil, err
		}
		// reconstruct the action response
		// Note that hiveOT uses correlationID instead of actionID
		// hiveot also doesn't return timerequested
		output := msg.ResponseMessage{
			Operation:     td.OpInvokeAction,
			ThingID:       wssResp.ThingID,
			Name:          wssResp.Name,
			Output:        wssStatus.Output,
			Status:        wssStatus.State,
			CorrelationID: wssResp.CorrelationID,
			Timestamp:     wssStatus.TimeEnded,
			Error:         wssResp.Error,
		}
		respMsg.Output = output

	case td.OpQueryAllActions:
		// WSS ResponseMessage contains ActionStatus map
		var wssStatusMap map[string]WotWssActionStatus
		output := make(map[string]msg.ResponseMessage)
		err = utils.Decode(wssResp.Statuses, &wssStatusMap)
		if err != nil {
			return nil, err
		}
		// reconstruct the latest responses for the actions
		for name, wssStatus := range wssStatusMap {
			output[name] = msg.ResponseMessage{
				Operation:     td.OpInvokeAction,
				ThingID:       wssResp.ThingID,
				Name:          name,
				Output:        wssStatus.Output,
				Status:        wssStatus.State,
				CorrelationID: wssResp.CorrelationID,
				Timestamp:     wssStatus.TimeEnded,
				Error:         wssResp.Error,
			}
		}
		respMsg.Output = output

	case td.OpReadAllProperties, td.OpReadMultipleProperties:
		// the 'Values' property from the msg.ResponseMessage embedded struct
		// contains the object with all property-value names
		respMsg.Output = wssResp.Values
	case td.OpReadProperty:
		// the 'Value' property contains the actual value
		respMsg.Output = wssResp.Value
	}
	return respMsg, err
}

// EncodeNotification converts a hiveot RequestMessage to a websocket equivalent message
func (svc *WotWssMsgEncoder) EncodeNotification(notif *msg.NotificationMessage) ([]byte, error) {

	op := td.OpSubscribeEvent
	if notif.AffordanceType == msg.AffordanceTypeProperty {
		op = td.OpObserveProperty
	}
	wssNotif := WotWssNotificationMessage{
		WotWssCommon: WotWssCommon{
			CorrelationID: notif.CorrelationID,
			MessageType:   msg.MessageTypeNotification,
			Name:          notif.Name,
			Operation:     op,
			ThingID:       notif.ThingID,
			Timestamp:     notif.Timestamp,
		},
		Data:  notif.Data, // in case of events
		Value: notif.Data, // in case of properties
	}

	return jsoniter.Marshal(wssNotif)
}

// EncodeRequest converts a hiveot RequestMessage to websocket equivalent message
func (svc *WotWssMsgEncoder) EncodeRequest(req *msg.RequestMessage) ([]byte, error) {
	wssReq := WotWssRequestMessage{
		WotWssCommon: WotWssCommon{
			CorrelationID: req.CorrelationID,
			MessageType:   msg.MessageTypeRequest,
			Name:          req.Name,
			Operation:     req.Operation,
			ThingID:       req.ThingID,
			Timestamp:     req.Timestamp,
		},
		// ActionID: req.CorrelationID, // correlationID is used as actionID // TBD: bother with this?
		Input:    req.Input, // in case of invokeaction
		SenderID: req.SenderID,
		// Names:     req.Input,  // in case of readmultiple/allproperties
		Value:  req.Input, // in case of writeproperty
		Values: req.Input, // in case of writeallproperties, writemultipleproperties
	}
	if req.Operation == td.OpReadAllProperties || req.Operation == td.OpReadMultipleProperties {
		names, ok := req.Input.([]string)
		if !ok {
			slog.Error("EncodeRequest: Input expected an array of names for op", "op", req.Operation)
		}
		wssReq.Names = names
	}
	return jsoniter.Marshal(wssReq)
}

// EncodeResponse converts a hiveot ResponseMessage to websocket equivalent message
// This always returns a response
func (svc *WotWssMsgEncoder) EncodeResponse(resp *msg.ResponseMessage) ([]byte, error) {
	var err error

	// the hiveot response message partially identical to the websocket message
	// only convert where they differ
	wssResp := WotWssResponseMessage{
		WotWssCommon: WotWssCommon{
			CorrelationID: resp.CorrelationID,
			MessageType:   msg.MessageTypeResponse,
			Name:          resp.Name,
			Operation:     resp.Operation,
			ThingID:       resp.ThingID,
			Timestamp:     resp.Timestamp,
		},
		// ActionID: resp.CorrelationID, // correlationID is used as actionID // TBD: bother with this?
		// the hiveot error response is identical to the websocket error response
		Error: resp.Error,
		// MessageID:   resp.MessageID,
		Output: resp.Output,
		// Status:   resp.Status,  needs to be an action status
		Statuses: nil,
		Values:   nil,
	}
	// last, set status(es) and values, depending on the operation
	switch resp.Operation {
	case td.OpCancelAction:
		// actionID of cancelled action ?
		// wssResp.ActionID = resp.CorrelationID
	case td.OpInvokeAction:
		wssResp.Status = &WotWssActionStatus{
			// websocket asynchronous response returns ActionID
			ActionID:  resp.CorrelationID,
			Error:     resp.Error, // error fields are identical
			State:     resp.Status,
			Output:    resp.Output,
			TimeEnded: resp.Timestamp,
		}
	case td.OpQueryAction:
		// the output is the last response: convert it to an ActionStatus object
		qResp := msg.ResponseMessage{}
		utils.Decode(resp.Output, &qResp)
		wssResp.Status = &WotWssActionStatus{
			ActionID:  qResp.CorrelationID,
			Error:     qResp.Error, // error fields are identical
			Output:    qResp.Output,
			State:     qResp.Status,
			TimeEnded: qResp.Timestamp,
		}
	case td.OpQueryAllActions:
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
				State:     respMsg.Status,
				TimeEnded: respMsg.Timestamp,
				Output:    respMsg.Output,
			}
		}
		wssResp.Statuses = wssStatusMap
	case td.OpReadAllProperties, td.OpReadMultipleProperties:
		// ReadAllProperties has the same response object with property key-values
		wssResp.Values = resp.Output
	case td.OpReadProperty:
		wssResp.Value = resp.Output
	}

	return jsoniter.Marshal(wssResp)
}

// GetProtocolType returns the hiveot WSS protocol type identifier
// func (svc *WotWssMsgConverter) GetProtocolType() string {
// 	return wssapi.WotWSSProtocolType
// }

// Create a new instance of the WoT websocket to hiveot message converter
func NewWotWssMsgEncoder() *WotWssMsgEncoder {
	return &WotWssMsgEncoder{}
}
