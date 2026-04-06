package internal

import (
	jsoniter "github.com/json-iterator/go"
)

// raw gRPC messaging codec.
//
// If the input is a byte array then it is a direct pass without conversion.
// If the input is a string then it is returned as a byte array without conversion.
// Any other data type is json encoded.
type JsonCodec struct {
}

func (c JsonCodec) Name() string {
	return "jsoncodec"
}

// This is a codec simply passes the data as a byte array.
//
// Input data must be a byte array or string
func (JsonCodec) Marshal(v interface{}) ([]byte, error) {
	if data, ok := v.([]byte); ok {
		return data, nil
	} else if data, ok := v.(string); ok {
		return []byte(data), nil
	}
	return jsoniter.Marshal(v)
}

func (JsonCodec) Unmarshal(data []byte, v interface{}) error {
	if out, ok := v.(*[]byte); ok {
		*out = data
		return nil
	} else if out, ok := v.(*string); ok {
		*out = string(data)
		return nil
	}
	return jsoniter.Unmarshal(data, v)
}

func (JsonCodec) String() string {
	return "jsoncodec"
}
