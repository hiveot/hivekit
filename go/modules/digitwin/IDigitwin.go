package digitwin

import (
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
)

// The default instance ID of the digital twin module
const DigitwinModuleType = "digitwin"

// the default digital twin service ID for handling digitwin requests
const DefaultDigitwinThingID = "digitwin"

// the prefix used for digital twins
const DigitwinIDPrefix = "dtw:"

// OnlinePropName is the digital twin property name indicating the device is reachable
const OnlinePropName = "online"

// IDigitwinService is the interface of the digitwin module
type IDigitwinService interface {
	modules.IHiveModule

	// Return the internal device directory
	// Intended for modules like the router, that need to connect to the devices themselves.
	// GetDeviceDirectory() directory.IDirectoryServer

	// Return the original device TD by its thingID
	GetDeviceTD(thingID string) *td.TD
}
