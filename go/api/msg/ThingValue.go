package msg

import (
	"time"

	"github.com/hiveot/hivekit/go/utils"
)

// Thing values for reading properties and events from Things.
//
// Timestamp availability depends on the underlying transport protocol.
// All HiveOT transports support the timestamp.
// WoT transports do not. In this case Timestamp is empty.
type ThingValue struct {
	// The raw data as decribed in the affordance
	Data any `json:"data"`
	// RFC3339 Time the value was updated
	Timestamp string `json:"timestamp,omitempty"`
}

// ToString is a helper to easily read the notification data as a string
func (tv *ThingValue) ToString(maxlen int) string {
	return utils.DecodeAsString(tv.Data, maxlen)
}

// GetTime returns the timestamp in the local timezone
// This returns the current time if the value doesnt have a timestamp
func (tv *ThingValue) GetTime() time.Time {
	if tv.Timestamp == "" {
		return time.Now()
	}
	ts, _ := utils.DecodeAsTime(tv.Timestamp)
	return ts
}
