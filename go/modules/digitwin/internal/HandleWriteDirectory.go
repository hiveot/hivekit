package internal

import (
	"fmt"

	digitwinapi "github.com/hiveot/hivekit/go/modules/digitwin/api"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hivekit/go/wot/vocab"
	jsoniter "github.com/json-iterator/go"
)

// Delete the digital twin TD of the thingID.
// This returns an error if the TD should not be deleted
func (m *DigitwinService) HandleDeleteTD(agentID string, thingID string) error {

	agentThingID := fmt.Sprint(agentID, ":", thingID)
	m.deviceTDBucket.Delete(agentThingID)
	return nil
}

// HandleWriteDirectory is invoked before a TD is updated in the directory.
// This is the registered callback handler of the Thing directory.
//
// This:
// 1. Ignores services - return them as-is
// 2. Stores the original TD in the 'device TD bucket'
// 3. Replace the ThingID with that of the digital twin
// 4. Add a 'online' property that is updated if the agent changes connection status
// 5. Removes all forms
// 6. Inserts forms that point to the digital twin
//
// This returns the updated TD, or the old one if no digital twin is used for this Thing.
func (m *DigitwinService) HandleWriteDirectory(agentID string, tdi *td.TD) (*td.TD, error) {

	// 1. service types do not get a digital twin
	// this seems a bit simplistic but it avoids hiveot modules from getting a twin
	if tdi.AtType == vocab.DeviceTypeService {
		return tdi, nil
	}

	// 2. store the original TD and its agent for retrieval by the router
	tdi.AgentID = agentID
	tdJson, _ := jsoniter.Marshal(tdi)
	m.deviceTDBucket.Set(tdi.ID, tdJson)

	// 3. change the device ID to the digitwin ID
	// note that this modifies the original TD. - is this a problem?
	dtwTD := tdi
	digitwinThingID := MakeDigitwinID(agentID, tdi.ID)
	dtwTD.ID = digitwinThingID

	// 4. add a 'online' property indicating if it is reachable
	dtwTD.AddProperty(digitwinapi.OnlinePropName,
		"Online", "Indicate if the Thing is reachable", td.DataTypeBool)

	// 5. reset all existing forms and auth info
	dtwTD.Forms = make([]td.Form, 0)
	dtwTD.Security = nil
	dtwTD.SecurityDefinitions = make(map[string]td.SecurityScheme)

	for _, aff := range dtwTD.Properties {
		aff.Forms = make([]td.Form, 0)
	}
	for _, aff := range dtwTD.Events {
		aff.Forms = make([]td.Form, 0)
	}
	for _, aff := range dtwTD.Actions {
		aff.Forms = make([]td.Form, 0)
	}

	// 6. populate the TD with forms and security definitions of the available transports
	if m.addForms != nil {
		m.addForms(dtwTD, m.includeAffordanceForms)
	}

	return dtwTD, nil
}
