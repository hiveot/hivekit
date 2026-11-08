package wot_test

import (
	"testing"

	"github.com/cstockton/go-conv"
	"github.com/hiveot/hivekit/go/api/wot"
	jsoniter "github.com/json-iterator/go"

	"github.com/stretchr/testify/assert"
)

// testing of marshalling and unmarshalling schemas

func TestStringSchema(t *testing.T) {
	ss := wot.DataSchema{
		Type:            wot.DataTypeString,
		StringMinLength: 10,
	}
	enc1, err := jsoniter.Marshal(ss)
	assert.NoError(t, err)
	//
	ds := wot.DataSchema{}
	err = jsoniter.Unmarshal(enc1, &ds)
	assert.NoError(t, err)
}

func TestObjectSchema(t *testing.T) {
	atType := "hiveot:complexType"
	os := wot.DataSchema{
		Type:       wot.DataTypeObject,
		Properties: make(map[string]*wot.DataSchema),
		AtType:     atType,
	}
	os.Properties["stringProp"] = &wot.DataSchema{
		Type:            wot.DataTypeString,
		StringMinLength: 10,
	}
	os.Properties["intProp"] = &wot.DataSchema{
		Type:    wot.DataTypeInteger,
		Minimum: 10,
		Maximum: 20,
	}
	enc1, err := jsoniter.Marshal(os)
	assert.NoError(t, err)
	var ds map[string]interface{}
	err = jsoniter.Unmarshal(enc1, &ds)
	assert.NoError(t, err)

	var as wot.DataSchema
	err = jsoniter.Unmarshal(enc1, &as)
	assert.NoError(t, err)

	minVal := as.Properties["intProp"].Minimum
	minInt, err := conv.Int(minVal)
	assert.Equal(t, 10, minInt)

	atType2 := as.GetAtTypeString()
	assert.Equal(t, atType, atType2)
}
