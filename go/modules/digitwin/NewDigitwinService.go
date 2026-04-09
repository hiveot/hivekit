package digitwin

import (
	"github.com/hiveot/hivekit/go/api/td"
	digitwinapi "github.com/hiveot/hivekit/go/modules/digitwin/api"
	"github.com/hiveot/hivekit/go/modules/digitwin/internal"
	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
)

// NewDigitwinService creates a new instance of the digital twin service module.
// This module needs the directory that will receive TD's from devices and are queried
// by consumers for available TDs.
// The module will substitute the TDs with the digital twin and substitute forms with
// those pointing to this module.
//
//	storageDir is the directory where the module stores its data
//	dirModule is the directory service to hook into to intercept writes, or "" for in-memory testing
//	addForms is the handler to invoke to add forms to a TD
func NewDigitwinService(storageDir string, dirModule directoryapi.IDirectoryServer,
	addForms func(tdi *td.TD, includeAffordances bool)) digitwinapi.IDigitwinServer {

	m := internal.NewDigitwinService(storageDir, dirModule, addForms)
	return m
}
