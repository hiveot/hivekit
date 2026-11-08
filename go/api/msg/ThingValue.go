package msg

import (
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/utils"
)

// Thing values for reading properties and events from Things.
//
// Timestamp availability depends on the underlying transport protocol.
// All HiveOT transports support the timestamp.
// WoT transports do not. In this case Timestamp is empty.
type ThingValue struct {
	// The raw data as decribed in the affordance
	Raw any `json:"data"`
	// RFC3339 Time the value was updated
	Timestamp string `json:"timestamp,omitempty"`
	// Unit of this value
	Unit string `json:"unit,omitempty"`
}

// Array returns the value as an array
// The result depends on the Schema type
//
//	array: returns array of values as describe ni the Schema
//	boolean: returns a single element true/false
//	bytes: return an array of bytes
//	int: returns a single element with integer
//	object: returns a single element with object
//	string: returns a single element with string
func (v ThingValue) Array() []interface{} {
	objArr := make([]interface{}, 0)
	err := utils.DecodeAsObject(v.Raw, &objArr)
	_ = err
	return objArr
}

func (v ThingValue) Boolean() bool {
	return utils.DecodeAsBool(v.Raw)
}

// GetTime returns the timestamp the value was last updated
// This returns the current time if the value doesnt have a timestamp
func (tv *ThingValue) GetTime() time.Time {
	if tv.Timestamp == "" {
		return time.Now()
	}
	ts, _ := utils.DecodeAsTime(tv.Timestamp)
	return ts
}

// Return the value as an integer
func (v ThingValue) Integer() int {
	return utils.DecodeAsInt(v.Raw)
}

// Map returns the value as a name-value map
// Returns nil if no data was provided.
func (v ThingValue) Map() map[string]interface{} {
	o := make(map[string]interface{})
	err := utils.DecodeAsObject(v.Raw, &o)
	if err != nil {
		slog.Error("Can't convert value to a map", "value", v.Raw)
	}
	return o
}

// Number returns the value as a float32
func (v ThingValue) Number() float32 {
	return utils.DecodeAsNumber(v.Raw)
}

// Object decodes the value to the given object.
func (v ThingValue) Object(output interface{}) error {
	return utils.DecodeAsObject(v.Raw, output)
}

// Text returns the text representation of the value
func (v ThingValue) Text() string {
	valueStr := utils.DecodeAsString(v.Raw, 0)
	return valueStr
}

// ToString returns the text representation of the value with a size limit
// and unit symbol.
func (v ThingValue) ToString(maxlen int) string {
	valueStr := utils.DecodeAsString(v.Raw, maxlen)
	if v.Unit != "" {
		unitClass, found := vocab.UnitClassesMap[v.Unit]
		if found {
			valueStr += " " + unitClass.Symbol
		}
	}
	return valueStr
}

// Create a new instance of a thing's value with the update time
func NewThingValue(raw any, timeStamp string, unit string) *ThingValue {
	tv := &ThingValue{
		Raw:       raw,
		Timestamp: timeStamp,
		Unit:      unit,
	}
	return tv
}
