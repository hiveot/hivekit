package msg

import (
	"time"

	"github.com/hiveot/hivekit/go/utils"
)

// type of affordances used in messaging
type AffordanceType string

const AffordanceTypeEvent AffordanceType = "event"
const AffordanceTypeProperty AffordanceType = "property"
const AffordanceTypeAction AffordanceType = "action"

// ResponseMessage, ActionStatus and ThingValue define the standardized messaging
// envelopes for handling responses.
// Each transport protocol bindings map this format to this specific format.

// ThingValue hold the message with its value.
type ThingValue struct {
	// Type of affordance this is a value of: AffordanceTypeProperty|Event|Action
	AffordanceType AffordanceType `json:"affordanceType"`

	// Output with Payload
	//
	// Data in format as described by the thing's affordance
	Data any `json:"data,omitempty"`

	// Name with affordance name
	//
	// Name of the affordance holding the value
	Name string `json:"name,omitempty"`

	// The sender of the notification or request
	SenderID string `json:"senderID,omitempty"`

	// ThingID with Thing ID
	//
	// Digital twin Thing ID
	ThingID string `json:"thingID,omitempty"`

	// Timestamp with Timestamp time
	//
	// Time the value was last updated
	Timestamp string `json:"timestamp,omitempty"`
}

// ToString is a helper to easily read the response output as a string
func (tv *ThingValue) ToString(maxlen int) string {
	return utils.DecodeAsString(tv.Data, maxlen)
}
func NewThingValue(senderID string, affordanceType AffordanceType, thingID, name string, data any, timestamp string) *ThingValue {
	tv := &ThingValue{
		AffordanceType: affordanceType,
		Data:           data,
		Name:           name,
		ThingID:        thingID,
		SenderID:       senderID,
		Timestamp:      timestamp,
	}
	if tv.Timestamp == "" {
		tv.Timestamp = utils.FormatUTCMilli(time.Now())
	}
	return tv
}
