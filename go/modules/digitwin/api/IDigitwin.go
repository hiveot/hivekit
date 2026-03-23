package digitwinapi

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/vocab"
	"github.com/hiveot/hivekit/go/wot/td"
)

// The default instance ID of the digital twin module
const DefaultDigitwinModuleID = "digitwin"

// the prefix used for digital twins
const DigitwinIDPrefix = "dtw:"

// Device types that are services do not get a digital twin
const DeviceTypeService = vocab.ThingService

// OnlinePropName is the digital twin property name indicating the device is reachable
const OnlinePropName = "online"

// IDigitwinServer is the interface of the digitwin module
type IDigitwinServer interface {
	modules.IHiveModule

	// Return the internal device directory
	// Intended for modules like the router, that need to connect to the devices themselves.
	// GetDeviceDirectory() directoryapi.IDirectoryServer

	// Return the original device TD by its thingID
	GetDeviceTD(thingID string) *td.TD
}
