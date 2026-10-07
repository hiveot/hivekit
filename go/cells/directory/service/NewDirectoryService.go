package directory_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/directory/service/internal"
)

// NewDirectoryService returns a ready-to-use Thing directory service instance.
// This opens or creates a directory in the provided storage directory.
//
// A hook to provide a list of transport servers can be used to add forms to TD's
// from local services and RC devices that are reachable via these servers.
// When updating these TD's the servers are invoked to update the forms.
//
//	thingID is the instance ID of the directory server. Use "" for default hiveot:directory.
//	location is the location where the service stores its data. Use "" for testing with an in-memory store.
//	addForms hook to provide available transport servers. nil to ignore.
func NewDirectoryService(
	thingID string, storageDir string, addForms func(*td.TD)) (directory.IDirectoryService, error) {

	svc, err := internal.NewDirectoryServiceImpl(thingID, storageDir, addForms)

	return svc, err
}

// Create the directory service instance using the factory environment
func NewDirectoryServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(directory.DirectoryServiceCellType)
	env.CreateDir(storageDir, 0700)

	thingID := ""
	svc, err := NewDirectoryService(thingID, storageDir, f.AddTDSecForms)
	return svc, err
}
