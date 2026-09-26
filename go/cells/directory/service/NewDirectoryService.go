package directory_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/directory/internal/serviceimpl"
)

// NewDirectoryService returns a ready-to-use Thing directory service instance.
// This opens or creates a directory in the provided storage directory.
//
// To expose the http API create the DirectoryHttpHandler and include it as the first transport
// in the list of transport. The first transport will be used as the base URL in the TDD.
//
//	thingID is the instance ID of the directory server. Use "" for default {hostname}:directory.
//	location is the location where the service stores its data. Use "" for testing with an in-memory store.
//	httpServer is used to expose the directory TDD on the well-known path.
func NewDirectoryService(
	thingID string, storageDir string, httpServer api.IHttpServer) (directory.IDirectoryService, error) {

	svc, err := serviceimpl.NewDirectoryServiceImpl(thingID, storageDir, httpServer)

	return svc, err
}

// Create the directory service instance using the factory environment
// The director http-service is optional. This will continue without http if the
// service is not yet loaded.
func NewDirectoryServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(directory.DirectoryServiceCellType)
	env.CreateDir(storageDir, 0700)

	httpServer := f.GetHttpServer(false)

	thingID := ""
	svc, err := NewDirectoryService(thingID, storageDir, httpServer)
	return svc, err
}
