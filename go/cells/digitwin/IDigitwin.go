package digitwin

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
)

// The default instance ID of the digital twin service
const DigitwinCellType = "digitwin"

// the default digital twin service (thing) ID for handling digitwin requests
const DefaultDigitwinServiceID = "digitwin"

// the prefix used for digital twins
const DigitwinIDPrefix = "dtw:"

// OnlinePropName is the digital twin property name indicating the device is reachable
const OnlinePropName = "online"

// IDigitwinService is the interface of the digitwin service
type IDigitwinService interface {
	api.IHiveCell

	// Return the internal device directory
	// Intended for cells like the router, that need to connect to the devices themselves.
	// GetDeviceDirectory() directory.IDirectoryServer

	// Return the original device TD by its thingID
	GetDeviceTD(thingID string) *td.TD
}
