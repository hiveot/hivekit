package msg

import (
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/utils"
)

// The status of an action. Also used as the result in QueryAction.
// See https://www.w3.org/TR/wot-profile/ for the ActionStatus object definition
type ActionStatus struct {
	// The correlationID/actionID of the request
	CorrelationID string `json:"correlationID,omitempty"`

	// An error message, if any, associated with a failed action.
	Error *wot.ErrorValue `json:"error,omitempty"`

	// The output data, if any, of a completed action which MUST conform with
	// the output data schema of the corresponding ActionAffordance.
	Output any `json:"output,omitempty"`

	// ID of the sender of the action
	// HiveOT extension to track the sender of a request.
	SenderID string `json:"senderID"`

	// The status of the action request. completed, failed
	Status string `json:"status"`

	// RFC3339 millisecond timestamp indicating the time at which the Thing successfully completed
	// status last updated. When status is completed this is the completion time.
	// If the action failed this is the failure time.
	TimeUpdated string `json:"timeEnded,omitempty"`

	// RFC3339 millisecond timestamp indicating the time at which the Thing received the request
	// to execute the action.
	TimeRequested string `json:"timeRequested,omitempty"`

	//--- HiveKit extension

	// Input for the action as described in the TD affordance dataschema.
	Input any `json:"input,omitempty"` // native
}

func NewActionStatus(input any, senderID string, correlationID string) *ActionStatus {
	as := ActionStatus{
		Status:        StatusPending,
		TimeRequested: utils.FormatNowUTCMilli(),
		Input:         input,
		SenderID:      senderID,
		CorrelationID: correlationID,
	}
	return &as
}
