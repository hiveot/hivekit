package utils

import (
	"log/slog"

	"github.com/cstockton/go-conv"
	jsoniter "github.com/json-iterator/go"
)

// Decode converts the any-type to the given interface type.
// If the output type is a native type then also consider using one of the DecodeAs...
// methods as these are likely more performant.
// This returns an error if conversion fails.
func Decode(value any, arg any) (err error) {

	if value == nil || arg == nil {
		arg = nil
		return nil
	}
	switch a := arg.(type) {
	case *bool:
		*a, err = conv.Bool(value)
	case *[]byte:
		s, err2 := conv.String(value)
		err = err2
		*a = []byte(s)
	case *string:
		*a, err = conv.String(value)
	case *int:
		*a, err = conv.Int(value)
	case *int16:
		*a, err = conv.Int16(value)
	case *int32:
		*a, err = conv.Int32(value)
	case *uint:
		*a, err = conv.Uint(value)
	case *uint32:
		*a, err = conv.Uint32(value)
	case *uint64:
		*a, err = conv.Uint64(value)
	case *float32:
		*a, err = conv.Float32(value)
	case *float64:
		*a, err = conv.Float64(value)
	default:
		// the ugly workaround is to marshal/unmarshal using json.
		// TODO: more efficient method to convert the any type to the given type.
		jsonData, _ := jsoniter.MarshalToString(value)
		err = jsoniter.UnmarshalFromString(jsonData, arg)
	}
	return err
}

// DecodeAsString converts the value to a string
// if value is already a string then it is returned as-is
// if maxlen is provided then limit the resulting length and add ... if exceeded. Use 0 for all.
func DecodeAsString(value any, maxlen int) string {
	asString, err := conv.String(value)
	if err != nil {
		return ""
	}
	if value == nil {
		return ""
	}
	// asString := ""
	// switch v2 := value.(type) {
	// case []byte:
	// 	asString = string(v2)
	// case string:
	// 	asString = v2
	// case *string:
	// 	asString = *v2
	// default:
	// 	asString = fmt.Sprintf("%v", value)
	// }
	if maxlen <= 0 || len(asString) <= maxlen {
		return asString
	}
	return asString[:maxlen-3] + "..."
}

// DecodeAsBool converts the value to a boolean.
// If value is already a boolean then it is returned as-is.
func DecodeAsBool(value any) bool {
	asBool, err := conv.Bool(value)
	if err != nil {
		slog.Warn("Can't convert value to a boolean", "value", value)
	}
	return asBool
}

// DecodeAsInt converts the value to an integer.
// This accepts int, int64, *int, bool, uint, float32/64
// If value is already an integer then it is returned as-is.
// If value > int (eg int64) then the result is unpredicable
func DecodeAsInt(value any) int {
	asInt, err := conv.Int(value)
	if err != nil {
		slog.Warn("Can't convert value to a integer", "value", value)
	}
	return asInt
}

// DecodeAsUInt converts the value to an unsigned integer.
// This accepts uint, uint64, *uint, bool, uint, float32/64
// If value is already an integer then it is returned as-is.
// If value > int (eg int64) then the result is unpredicable
func DecodeAsUint(value any) uint {
	asUint, err := conv.Uint(value)
	if err != nil {
		slog.Warn("Can't convert value to a integer", "value", value)
	}
	return asUint
}

// DecodeAsNumber converts the value to a float32 number.
// If value is already a float32 then it is returned as-is.
func DecodeAsNumber(value any) float32 {
	asFloat32, err := conv.Float32(value)
	if err != nil {
		slog.Warn("Can't convert value to a float", "value", value)
	}
	return asFloat32
}

// DecodeAsObject converts the value to an object.
// If the object is of the same type then it is copied
// otherwise a json marshal/unmarshal is attempted for a deep conversion.
func DecodeAsObject(value any, object interface{}) error {
	if value == nil {
		object = nil
		return nil
	} else {
		serObj, err := jsoniter.Marshal(value)
		if err == nil {
			err = jsoniter.Unmarshal(serObj, object)
		}
		return err
	}
}
