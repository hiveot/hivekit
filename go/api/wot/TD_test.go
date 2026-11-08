package wot_test

import (
	"strings"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

const deviceTypeThingSensor = "sensor"
const propNameDeviceSoftwareVersion = "softwareVersion"
const unitNamePercent = "%"

const thing1ID = "thing1"

func TestCreateTD(t *testing.T) {
	tdoc := wot.NewTD(thing1ID, "test TD", deviceTypeThingSensor)
	assert.NotNil(t, tdoc)

	// Set version
	//versions := map[string]string{"Software": "v10.1", "Hardware": "v2.0"}
	propAffordance := &wot.PropertyAffordance{
		DataSchema: wot.DataSchema{
			Type:  wot.DataTypeArray,
			Title: "version",
		},
	}
	tdoc.UpdateProperty(propNameDeviceSoftwareVersion, propAffordance)

	// Define TD property
	propAffordance = &wot.PropertyAffordance{
		DataSchema: wot.DataSchema{
			Type: wot.DataTypeString,
			Enum: make([]interface{}, 0), //{"value1", "value2"},
			Unit: "C",
		},
	}
	//propAffordance.SetOneOfValues([]string{})

	thingType := tdoc.AtType
	assert.Equal(t, deviceTypeThingSensor, thingType)

	// created time must be set to RFC3339
	assert.NotEmpty(t, tdoc.Created)
	t1, err := time.Parse(time.RFC3339, tdoc.Created)
	assert.NoError(t, err)
	assert.NotNil(t, t1)

	tdoc.UpdateProperty("prop1", propAffordance)
	prop := tdoc.GetProperty("prop1")
	assert.NotNil(t, prop)

	tdoc.UpdateTitleDescription("title", "description")

	tdoc.UpdateAction("action1", &wot.ActionAffordance{})
	action := tdoc.GetAction("action1")
	assert.NotNil(t, action)

	tdoc.UpdateEvent("event1", &wot.EventAffordance{})
	ev := tdoc.GetEvent("event1")
	assert.NotNil(t, ev)

	tdoc.SetForms([]wot.Form{})

	asMap := tdoc.AsMap()
	assert.NotNil(t, asMap)
}

func TestMissingAffordance(t *testing.T) {

	// test return nil if no affordance is found
	tdoc := wot.NewTD(thing1ID, "test TD", deviceTypeThingSensor)
	assert.NotNil(t, tdoc)

	prop := tdoc.GetProperty("prop1")
	assert.Nil(t, prop)

	action := tdoc.GetAction("action1")
	assert.Nil(t, action)

	ev := tdoc.GetEvent("event1")
	assert.Nil(t, ev)
}

func TestAtType(t *testing.T) {
	tdoc := wot.NewTD(thing1ID, "test TD", deviceTypeThingSensor)
	at1 := tdoc.GetAtTypes()
	assert.Equal(t, deviceTypeThingSensor, at1[0])
	tdoc.SetType(vocab.DeviceActuator)

	at2 := tdoc.GetAtTypes()
	assert.IsType(t, []string{}, at2)
	assert.Len(t, at2, 2)
}

func TestAddProp(t *testing.T) {
	prop2AtType := "test:specialtype"
	tdoc := wot.NewTD(thing1ID, "test TD", deviceTypeThingSensor)
	tdoc.AddProperty("prop1", "prop 1", "test property", wot.DataTypeBool)

	aff := tdoc.AddProperty("prop2", "test property2", "", wot.DataTypeString)
	aff.SetAtType(prop2AtType)
	aff.Unit = unitNamePercent

	// retrieve a property by its @type value
	pa2Name, pa2 := tdoc.GetPropertyOfVocabType(prop2AtType)
	assert.Equal(t, "prop2", pa2Name)
	assert.NotNil(t, pa2)

	prop := tdoc.GetProperty("prop1")
	assert.NotNil(t, prop)
	time.Sleep(time.Millisecond)
	prop = tdoc.GetProperty("prop2")
	assert.NotNil(t, prop)

	tdoc.AddPropertyAsBool("b1", "bool1", "boolean 1")
	tdoc.AddPropertyAsString("s1", "string1", "string 1")
	tdoc.AddPropertyAsInt("i1", "int1", "integer 1")

}

// test that IDs with spaces are escaped
func TestAddPropBadIDs(t *testing.T) {
	propID := "prop 1"
	tdoc := wot.NewTD(thing1ID, "test TD", deviceTypeThingSensor)
	tdoc.AddProperty(propID, "test property", "", wot.DataTypeBool)

	tdoc.AddProperty("prop2", "test property2", "", wot.DataTypeString)

	prop := tdoc.GetProperty(propID)
	assert.Nil(t, prop)
	prop = tdoc.GetProperty(strings.ReplaceAll(propID, " ", "_"))
	assert.NotNil(t, prop)

	time.Sleep(time.Millisecond)
	prop = tdoc.GetProperty("prop2")
	assert.NotNil(t, prop)
}

func TestAddEvent(t *testing.T) {
	tdoc := wot.NewTD(thing1ID, "test TD", deviceTypeThingSensor)
	tdoc.AddEvent("event1", "Test Event", "", nil)

	tdoc.AddEvent("event2", "Test Event", "", nil)

	ev := tdoc.GetEvent("event1")
	assert.NotNil(t, ev)
	time.Sleep(time.Millisecond)
	ev = tdoc.GetEvent("event2")
	assert.NotNil(t, ev)
}

func TestAddAction(t *testing.T) {
	tdoc := wot.NewTD(thing1ID, "test TD", deviceTypeThingSensor)
	tdoc.AddAction("action1", "test", "Test Action", nil)

	// has a space
	tdoc.AddAction("action 2", "test", "test Action", nil)
	tdoc.EscapeKeys()
	action := tdoc.GetAction("action1")
	assert.NotNil(t, action)
	time.Sleep(time.Millisecond)
	action = tdoc.GetAction("action_2")
	assert.NotNil(t, action)
}

// just some basic tests. need much more
func TestForms(t *testing.T) {
	const action1Name = "action1"
	const prop1Name = "prop1"
	const event1Name = "event1"
	tdoc := wot.NewTD(thing1ID, "test TD", deviceTypeThingSensor)
	tdoc.Base = "https://localhost:1234/path"

	// add actions
	actAff := tdoc.AddAction(action1Name, "action", "Test Action", nil)
	tdoc.AddProperty(prop1Name, "prop", "Test Prop", wot.DataTypeInteger)
	tdoc.AddEvent(event1Name, "event", "Test Event", nil)

	actForm := wot.NewForm(wot.OpInvokeAction, "https://localhost/action")
	actAff.Forms = []wot.Form{actForm}

	forms := make([]wot.Form, 0)
	//forms = append(forms, wot.NewForm(wot.OpInvokeAction, "https://localhost/action"))
	forms = append(forms, wot.NewForm(wot.OpWriteProperty, "https://localhost/prop"))
	forms = append(forms, wot.NewForm(wot.OpSubscribeEvent, "https://localhost/ev"))
	tdoc.AddForms(forms)

	//
	f1 := tdoc.GetForms(wot.OpWriteProperty, prop1Name)
	require.NotNil(t, f1)
	f2 := tdoc.GetForms(wot.OpInvokeAction, action1Name)
	require.NotNil(t, f2)
	f3 := tdoc.GetForms(wot.OpSubscribeEvent, event1Name)
	require.NotNil(t, f3)

	f1b, match := tdoc.GetForm(
		wot.OpWriteProperty, prop1Name,
		api.WotHttpBasicScheme, api.WotHttpBasicSubprotocol)
	require.NotEmpty(t, f1b)
	assert.True(t, match)

	href, found := f1b.GetHRef()
	assert.True(t, found)
	assert.NotEmpty(t, href)

	// the form href is absolute
	hrefURL, err := f1b.ResolveHRef(tdoc.Base, nil)
	assert.NoError(t, err)
	assert.Equal(t, href, hrefURL.String())
}

// test adding security scheme
func TestSecScheme(t *testing.T) {

	tdoc := wot.NewTD(thing1ID, "test TD", deviceTypeThingSensor)
	tdoc.Base = "https://localhost:1234/path"

	// add security
	secScheme1 := wot.SecurityScheme{
		Scheme: "bearer",
		In:     "header",
	}
	tdoc.AddSecurityScheme("bearer", secScheme1)
	assert.Len(t, tdoc.SecurityDefinitions, 1)
	scheme1, hasMultiple, err := tdoc.GetSecurityScheme()
	require.NoError(t, err)
	require.NotEmpty(t, scheme1)
	assert.False(t, hasMultiple)

	// no duplicates
	tdoc.AddSecurityScheme("bearer", secScheme1)
	assert.Len(t, tdoc.SecurityDefinitions, 1)
	scheme1, hasMultiple, err = tdoc.GetSecurityScheme()
	require.NoError(t, err)
	require.NotEmpty(t, scheme1)
	assert.False(t, hasMultiple)

	securityValue, ok := tdoc.Security.(string)
	assert.True(t, ok, "expected a string for security")
	assert.Equal(t, "bearer", securityValue)

	// add a second scheme
	secScheme2 := wot.SecurityScheme{
		Scheme: "basic",
		In:     "header",
	}
	tdoc.AddSecurityScheme("basic_sc", secScheme2)
	assert.Len(t, tdoc.SecurityDefinitions, 2)
	scheme2, hasMultiple, err := tdoc.GetSecurityScheme()
	require.NoError(t, err)
	require.NotEmpty(t, scheme2)
	assert.True(t, hasMultiple)

}
