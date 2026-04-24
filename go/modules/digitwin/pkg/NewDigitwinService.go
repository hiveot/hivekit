package digitwinpkg

import (
	"path/filepath"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/digitwin"
	"github.com/hiveot/hivekit/go/modules/digitwin/internal"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/factory"
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
func NewDigitwinService(storageDir string, dirModule directory.IDirectoryService,
	addForms func(tdi *td.TD, includeAffordances bool)) digitwin.IDigitwinService {

	m := internal.NewDigitwinService(storageDir, dirModule, addForms)
	return m
}

// Create a new digitwin service using the module factory
// This loads the directory module and hooks itself into it to intercept directory writes.
func NewDigitwinServiceFactory(f factory.IModuleFactory) modules.IHiveModule {
	env := f.GetEnvironment()

	// data is stored in a module subdir
	storageDir := filepath.Join(env.StoresDir, digitwin.DigitwinModuleType)

	// the directory module used to intercept directory writes to create digital twins of
	m, err := f.GetModule(directory.DirectoryModuleType, true)
	if err != nil {
		return nil
	}
	dirModule, ok := m.(directory.IDirectoryService)
	_ = ok
	m = NewDigitwinService(storageDir, dirModule, f.AddTDSecForms)
	return m
}
