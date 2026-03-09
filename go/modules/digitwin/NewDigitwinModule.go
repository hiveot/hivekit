package digitwin

import (
	digitwinapi "github.com/hiveot/hivekit/go/modules/digitwin/api"
	"github.com/hiveot/hivekit/go/modules/digitwin/internal/module"
	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
	"github.com/hiveot/hivekit/go/wot/td"
)

// Create a new instance of the digital twin module.
// This module needs the directory that will receive TD's from devices and are queried
// by consumers for available TDs.
// The module will substitute the TDs with the digital twin and substitute forms with
// those pointing to this module.
//
//		storageRoot is the root directory where modules store their data into a moduleID subdirectory
//		dirModule is the directory service to hook into to intercept writes, or "" for in-memory testing
//	 addForms is the handler to invoke to add forms to a TD
func NewDigitwinModule(storageRoot string, dirModule directoryapi.IDirectoryServer,
	addForms func(tdi *td.TD, includeAffordances bool)) digitwinapi.IDigitwinModule {

	m := module.NewDigitwinModule(storageRoot, dirModule, addForms)
	return m
}
