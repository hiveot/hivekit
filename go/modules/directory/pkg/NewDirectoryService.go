package directorypkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/directory/internal"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
)

const DirectoryModuleType = directory.DirectoryModuleType

// NewDirectoryService creates a new Thing directory service module instance.
// On start this opens or creates a directory in the provided storage directory.
//
// If a http server is provided this registers the HTTP API with the router and serves
// its TD on the .well-known/wot endpoint as per discovery specification.
//
//	serviceID is the directory service instance thingID, use "" for the default.
//	storageDir is the location where the module stores its data. Use "" for testing with an in-memory store.
//	httpServer optional http server to register the html API handlers with. nil to ignore.
func NewDirectoryService(serviceID string, storageDir string, httpServer transports.IHttpServer) directory.IDirectoryServer {
	m := internal.NewDirectoryService(serviceID, storageDir, httpServer)
	return m
}

// Create the directory service module using the factory environment
func NewDirectoryServiceFactory(f factory.IModuleFactory) modules.IHiveModule {
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(directory.DirectoryModuleType)
	// FIXME: how to configure use of the http server and the directory instance ID?
	httpServer := f.GetHttpServer()

	m := NewDirectoryService(directory.DefaultDirectoryThingID, storageDir, httpServer)
	return m
}
