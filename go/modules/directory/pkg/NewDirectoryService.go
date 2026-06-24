package directorypkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/directory/internal/service"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transport"
)

// NewDirectoryService creates a new Thing directory server module instance.
// On start this opens or creates a directory in the provided storage directory.
//
// To expose the http API create the DirectoryHttpHandler module and include it
// as the first transport in the list of transport. The first transport will be used
// as the base URL in the TDD.
//
//	thingID is the instance ID of the directory server. Use "" for default
//	location is the location where the module stores its data. Use "" for testing with an in-memory store.
//	httpServer is used to expose the directory TDD on the well-known path.
//	transports is a list of transports that should be included in the TDD security and forms
func NewDirectoryService(
	thingID string, storageDir string, httpServer transport.IHttpServer,
	transports []transport.ITransportServer) directory.IDirectoryService {

	m := service.NewDirectoryServiceImpl(
		thingID, storageDir, httpServer, transports)

	return m
}

// Create the directory service module using the factory environment
// The director http-service is optional. This will continue without http if the
// module is not yet loaded.
func NewDirectoryServiceFactory(f factory.IModuleFactory, md *factory.ModuleDefinition) (modules.IHiveModule, error) {
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(directory.DirectoryServiceModuleType)

	// httpMod, _ := f.GetModule(directory.DirectoryHttpModuleType, false)
	// httpAPI, ok := httpMod.(directory.IDirectoryHttpServer)
	// if !ok {
	// 	slog.Info("NewDirectoryMsgServerFactory: No http so running directory without http api")
	// }
	transports := f.GetTransportServers()

	m := NewDirectoryService("", storageDir, f.GetHttpServer(false), transports)
	return m, nil
}
