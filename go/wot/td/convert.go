package td

import (
	"errors"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/araddon/dateparse"
	"github.com/hiveot/hivekit/go/wot"
	jsoniter "github.com/json-iterator/go"
)

// Helper methods for converting TD, event, property and action values to and from text.
// Intended for assisting conversion between text and native formats.

// UnmarshalTD unmarshals a JSON encoded TD
func MarshalTD(tdi *TD) (tdJson string, err error) {
	tdJsonRaw, err := jsoniter.Marshal(tdi)
	return string(tdJsonRaw), err
}

// UnmarshalTD unmarshals a JSON encoded TD
func UnmarshalTD(tdJSON string) (tdi *TD, err error) {
	tdi = &TD{}
	err = jsoniter.UnmarshalFromString(tdJSON, tdi)
	return tdi, err
}

// ReadTD reads and unmarshals a JSON encoded TD from the given reader
func ReadTD(r io.Reader) (tdi *TD, err error) {
	tdJsonRaw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	tdi = &TD{}
	err = jsoniter.Unmarshal(tdJsonRaw, tdi)
	return tdi, err
}

// UnmarshalTDList unmarshals a list of JSON encoded TDs

func UnmarshalTDList(tdListJSON []string) (tdList []*TD, err error) {
	tdList = make([]*TD, 0, len(tdListJSON))
	for _, tdJson := range tdListJSON {
		tdi := TD{}
		err = jsoniter.UnmarshalFromString(tdJson, &tdi)
		if err == nil {
			tdList = append(tdList, &tdi)
		}
	}
	return tdList, err
}

// ConvertToNative converts the string value to native type based on the given data schema
// this converts int, float, and boolean
// if the dataschema is an object or an array then strVal is assumed to be json encoded
func ConvertToNative(strVal string, dataSchema *DataSchema) (val any, err error) {
	if strVal == "" {
		// nil value boolean input are always treated as false.
		if dataSchema.Type == wot.DataTypeBool {
			return false, nil
		}
		return nil, nil
	} else if dataSchema == nil {
		slog.Error("ConvertToNative: nil DataSchema")
		return nil, errors.New("nil DataSchema")
	}
	switch dataSchema.Type {
	case wot.DataTypeBool:
		// ParseBool is too restrictive
		lowerVal := strings.ToLower(strVal)
		val = false
		if strVal == "1" || lowerVal == "true" || lowerVal == "on" {
			val = true
		}
	case wot.DataTypeArray:
		err = jsoniter.UnmarshalFromString(strVal, &val)
	case wot.DataTypeDateTime:
		val, err = dateparse.ParseAny(strVal)
	case wot.DataTypeInteger:
		val, err = strconv.ParseInt(strVal, 10, 64)
	case wot.DataTypeNumber:
		val, err = strconv.ParseFloat(strVal, 64)
	case wot.DataTypeUnsignedInt:
		val, err = strconv.ParseUint(strVal, 10, 64)
	case wot.DataTypeObject:
		err = jsoniter.UnmarshalFromString(strVal, &val)
	default:
		val = strVal
	}
	return val, err
}
