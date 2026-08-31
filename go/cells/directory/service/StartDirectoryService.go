package directory_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/directory/internal/serviceimpl"
)

// StartDirectoryService starts a new Thing directory service instance.
// This opens or creates a directory in the provided storage directory.
//
// To expose the http API create the DirectoryHttpHandler and include it as the first transport
// in the list of transport. The first transport will be used as the base URL in the TDD.
//
//	thingID is the instance ID of the directory server. Use "" for default
//	location is the location where the service stores its data. Use "" for testing with an in-memory store.
//	httpServer is used to expose the directory TDD on the well-known path.
//	transports is a list of transports that should be included in the TDD security and forms
func StartDirectoryService(
	thingID string, storageDir string, httpServer api.IHttpServer,
	transports []api.ITransportServer) (directory.IDirectoryService, error) {

	svc, err := serviceimpl.StartDirectoryServiceImpl(
		thingID, storageDir, httpServer, transports)

	return svc, err
}

// Create the directory service instance using the factory environment
// The director http-service is optional. This will continue without http if the
// service is not yet loaded.
func StartDirectoryServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(directory.DirectoryServiceCellType)
	env.CreateDir(storageDir, 0700)

	// httpMod, _ := f.GetCell(directory.DirectoryHttpServiceType, false)
	// httpAPI, ok := httpMod.(directory.IDirectoryHttpServer)
	// if !ok {
	// 	slog.Info("NewDirectoryMsgServerFactory: No http so running directory without http api")
	// }
	httpServer := f.GetHttpServer(false)
	transportMods := f.GetTransportServers()

	cellID := env.AppID + ":" + directory.DirectoryServiceCellType
	svc, err := StartDirectoryService(cellID, storageDir, httpServer, transportMods)
	return svc, err
}
