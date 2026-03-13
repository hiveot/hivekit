package digitwinapi

import (
	"github.com/hiveot/hivekit/go/modules"
	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
	"github.com/hiveot/hivekit/go/vocab"
)

// The default instance ID of the digital twin module
const DefaultDigitwinModuleID = "digitwin"

// the prefix used for digital twins
const DigitwinIDPrefix = "dtw:"

// Device types that are services do not get a digital twin
const DeviceTypeService = vocab.ThingService

// IDigitwinModule is the interface of the digitwin module
type IDigitwinModule interface {
	modules.IHiveModule

	// Return the internal device directory
	// Intended for modules like the router, that need to connect to the devices themselves.
	GetDeviceDirectory() directoryapi.IDirectoryServer
}
