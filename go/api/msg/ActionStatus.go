package msg

// The status of an action. Also used as the result in QueryAction.
// See https://www.w3.org/TR/wot-profile/ for the ActionStatus object definition
type ActionStatus struct {
	// The correlationID/actionID of the request
	CorrelationID string `json:"correlationID,omitempty"`

	// An error message, if any, associated with a failed action.
	Error *ErrorValue `json:"error,omitempty"`

	// The output data, if any, of a completed action which MUST conform with
	// the output data schema of the corresponding ActionAffordance.
	Output any `json:"output,omitempty"`

	// The status of the action request.
	Status string `json:"status"`

	// RFC3339 millisecond timestamp indicating the time at which the Thing successfully completed
	// executing the action, or failed to execute the action.
	TimeEnded string `json:"timeEnded,omitempty"`

	// RFC3339 millisecond timestamp indicating the time at which the Thing received the request
	// to execute the action.
	TimeRequested string `json:"timeRequested,omitempty"`

	//--- HiveKit extension

	// Input for the action as described in the TD affordance dataschema.
	Input any `json:"input,omitempty"` // native
}
