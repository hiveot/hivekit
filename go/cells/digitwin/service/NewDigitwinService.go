package digitwin_service

import (
	"fmt"
	"path/filepath"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/digitwin"
	"github.com/hiveot/hivekit/go/cells/digitwin/internal"
	"github.com/hiveot/hivekit/go/cells/directory"
)

// NewDigitwinService creates a new instance of the digital twin service.
// This service needs the directory that will receive TD's from devices and are queried
// by consumers for available TDs.
// The service will substitute the TDs with the digital twin and substitute forms with
// those pointing to this service.
//
//	storageDir is the directory where the service stores its data
//	dirService is the directory service to hook into to intercept writes, or "" for in-memory testing
//	addForms is the handler to invoke to add forms to a TD
func NewDigitwinService(storageDir string, dirSvc directory.IDirectoryService,
	addForms func(tdi *td.TD, includeAffordances bool)) digitwin.IDigitwinService {

	svc := internal.NewDigitwinServiceImpl(storageDir, dirSvc, addForms)
	return svc
}

// Create a new digitwin service using the factory
// This loads the directory service and hooks itself into it to intercept directory writes.
func NewDigitwinServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	env := f.GetEnvironment()

	// data is stored in a subdir
	storageDir := filepath.Join(env.StoresDir, digitwin.DigitwinCellType)

	// the directory used to intercept directory writes to create digital twins of
	svc, err := f.StartCell(directory.DirectoryServiceCellType, true)
	if err != nil {
		return nil, err
	}
	dirSvc, ok := svc.(directory.IDirectoryService)
	if !ok {
		return nil, fmt.Errorf("NewDigitwinServiceFactory: directory cell is wrong type")
	}
	svc = NewDigitwinService(storageDir, dirSvc, f.AddTDSecForms)
	return svc, nil
}
