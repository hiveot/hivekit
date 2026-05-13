package utils

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeToString(t *testing.T) {

	in1 := 42
	in2 := 42.5
	var out string
	var out2 string
	var out3 string

	out = DecodeAsString(in1, 0)
	assert.Equal(t, "42", out)

	err := Decode(in2, &out2)
	assert.NoError(t, err)
	assert.Equal(t, "42.5", out2)

	err = Decode(in2, &out3)
	assert.NoError(t, err)
	assert.Equal(t, "42.5", out3)
}

func TestDecodeToInt(t *testing.T) {

	in1 := "42"
	in2 := 42.5
	in3 := true
	var out int

	out = DecodeAsInt(in1)
	assert.Equal(t, 42, out)

	err := Decode(in1, &out)
	require.NoError(t, err)
	assert.Equal(t, 42, out)

	err = Decode(in2, &out)
	require.NoError(t, err)
	assert.Equal(t, 42, out)

	err = Decode(in3, &out)
	require.NoError(t, err)
	assert.Equal(t, 1, out)
}

func TestDecodeToFloat(t *testing.T) {

	in1 := "42.5"
	in2 := 42
	var out float32

	out = DecodeAsNumber(in1)
	assert.Equal(t, float32(42.5), out)

	err := Decode(in1, &out)
	require.NoError(t, err)
	assert.Equal(t, float32(42.5), out)

	err = Decode(in2, &out)
	require.NoError(t, err)
	assert.Equal(t, float32(42), out)
}

func TestDecodeToBool(t *testing.T) {

	in1 := "true"
	in2 := 1
	in3 := "1"
	var out bool

	out = DecodeAsBool(in1)
	assert.Equal(t, true, out)

	out = DecodeAsBool(in2)
	assert.Equal(t, true, out)

	out = DecodeAsBool(in3)
	assert.Equal(t, true, out)
}

func TestDecodeToObject(t *testing.T) {
	type AbcType struct {
		ValInt    int
		ValFloat  float32
		ValString string
	}

	a := AbcType{ValInt: 42, ValFloat: 32.5, ValString: "text"}

	// after (de)serialization the conversion should work
	var b any
	aSer, _ := jsoniter.Marshal(a)
	err := jsoniter.Unmarshal(aSer, &b)
	require.NoError(t, err)

	var c AbcType
	err = Decode(b, &c)
	require.NoError(t, err)
	assert.Equal(t, a.ValInt, c.ValInt)
	assert.Equal(t, a.ValFloat, c.ValFloat)
	assert.Equal(t, a.ValString, c.ValString)
}
