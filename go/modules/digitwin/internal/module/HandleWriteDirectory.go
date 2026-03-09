package module

import (
	"fmt"

	digitwinapi "github.com/hiveot/hivekit/go/modules/digitwin/api"
	"github.com/hiveot/hivekit/go/wot/td"
	jsoniter "github.com/json-iterator/go"
)

// Delete the digital twin TD of the thingID.
// This returns an error if the TD should not be deleted
func (m *DigitwinModule) HandleDeleteTD(agentID string, thingID string) error {

	agentThingID := fmt.Sprint(agentID, ":", thingID)
	m.bucket.Delete(agentThingID)
	return nil
}

// HandleWriteDirectory is invoked before a TD is updated in the directory.
//
// This filters TD's with the @type empty or set to service.
// TODO: should @type device be used instead?
//
// If the TD is that of a service then do nothing, otherwise update a digital twin.
//
// This returns the updated TD, or the old one if no digital twin is used for this Thing.
func (m *DigitwinModule) HandleWriteDirectory(agentID string, tdi *td.TD) (*td.TD, error) {

	// service types do not get a digital twin
	// this seems a bit simplistic but it avoids hiveot modules from getting a twin
	if tdi.AtType == digitwinapi.DeviceTypeService {
		return tdi, nil
	}

	// store the original TD's under the digitwin ID for easy retrieval
	digitwinThingID := CreateDigitwinID(agentID, tdi.ID)
	tdJson, _ := jsoniter.Marshal(tdi)
	m.bucket.Set(digitwinThingID, tdJson)

	// 1. change the device ID to the digitwin ID
	// note that this modifies the original TD. - is this a problem?
	dtwTD := tdi
	dtwTD.ID = digitwinThingID

	// 2. reset all existing forms and auth info
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

	// 3. populate the TD with forms and security definitions of the available transports
	if m.addForms != nil {
		m.addForms(dtwTD, m.includeAffordanceForms)
	}

	return dtwTD, nil
}
