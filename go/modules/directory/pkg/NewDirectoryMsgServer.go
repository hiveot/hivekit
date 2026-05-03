package directorypkg

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	internal "github.com/hiveot/hivekit/go/modules/directory/internal/msgserver"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// NewDirectoryServer creates a new Thing directory server module instance.
// On start this opens or creates a directory in the provided storage directory.
//
// To expose the http API create the DirectoryHttpHandler module and include it
// as the first transport in the list of transports. The first transport will be used
// as the base URL in the TDD.
//
//	thingID is the instance ID of the directory server. Use "" for default
//	location is the location where the module stores its data. Use "" for testing with an in-memory store.
//	httpAPI provides the security scheme and forms for the directory http endpoints. nil to not include these.
//	transports is a list of transports that should be included in the TDD security and forms
func NewDirectoryMsgServer(
	serviceID string, storageDir string, httpAPI directory.IDirectoryHttpServer,
	transports []transports.ITransportServer) directory.IDirectoryService {

	m := internal.NewDirectoryServer(
		serviceID, storageDir, httpAPI, transports)
	return m
}

// Create the directory service module using the factory environment
// The director http-service is optional. This will continue without http if the
// module is not yet loaded.
func NewDirectoryMsgServerFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(directory.DirectoryModuleType)

	// provide the directory http module instance for inclusing as the TDD base
	// this is optional. instantiate is false
	httpMod, _ := f.GetModule(directory.DirectoryHttpModuleType, false)
	httpAPI, ok := httpMod.(directory.IDirectoryHttpServer)
	if !ok {
		slog.Info("NewDirectoryMsgServerFactory: No http so running directory without http api")
	}
	transports := f.GetTransportServers()

	m := NewDirectoryMsgServer("", storageDir, httpAPI, transports)
	return m, nil
}
