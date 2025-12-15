package td_test

import (
	"testing"

	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	jsoniter "github.com/json-iterator/go"

	"github.com/stretchr/testify/assert"
)

// testing of marshalling and unmarshalling schemas

func TestStringSchema(t *testing.T) {
	ss := td.DataSchema{
		Type:            wot.DataTypeString,
		StringMinLength: 10,
	}
	enc1, err := jsoniter.Marshal(ss)
	assert.NoError(t, err)
	//
	ds := td.DataSchema{}
	err = jsoniter.Unmarshal(enc1, &ds)
	assert.NoError(t, err)
}

func TestObjectSchema(t *testing.T) {
	atType := "hiveot:complexType"
	os := td.DataSchema{
		Type:       wot.DataTypeObject,
		Properties: make(map[string]*td.DataSchema),
		AtType:     atType,
	}
	os.Properties["stringProp"] = &td.DataSchema{
		Type:            wot.DataTypeString,
		StringMinLength: 10,
	}
	os.Properties["intProp"] = &td.DataSchema{
		Type:    wot.DataTypeInteger,
		Minimum: 10,
		Maximum: 20,
	}
	enc1, err := jsoniter.Marshal(os)
	assert.NoError(t, err)
	var ds map[string]interface{}
	err = jsoniter.Unmarshal(enc1, &ds)
	assert.NoError(t, err)

	var as td.DataSchema
	err = jsoniter.Unmarshal(enc1, &as)
	assert.NoError(t, err)

	assert.Equal(t, 10, int(as.Properties["intProp"].Minimum))

	atType2 := as.GetAtTypeString()
	assert.Equal(t, atType, atType2)
}
