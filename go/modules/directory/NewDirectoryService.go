package directory

import (
	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules"
	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
	"github.com/hiveot/hivekit/go/modules/directory/internal"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// NewDirectoryService creates a new Thing directory service module instance.
// On start this opens or creates a directory in the provided storage directory.
//
// If a http server is provided this registers the HTTP API with the router and serves
// its TD on the .well-known/wot endpoint as per discovery specification.
//
// storageDir is the location where the module stores its data. Use "" for testing with an in-memory store.
// httpServer optional http server to register the html API handlers with. nil to ignore.
func NewDirectoryService(storageDir string, httpServer transports.IHttpServer) directoryapi.IDirectoryServer {
	m := internal.NewDirectoryService(storageDir, httpServer)
	return m
}

// Create the directory service module using the factory environment
func NewDirectoryServiceFactory(f factoryapi.IModuleFactory) modules.IHiveModule {
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(directoryapi.DirectoryModuleType)
	// FIXME: how to configure use of the http server?
	httpServer := f.GetHttpServer()

	m := internal.NewDirectoryService(storageDir, httpServer)
	return m
}
