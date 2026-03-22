package directory

import (
	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
	"github.com/hiveot/hivekit/go/modules/directory/internal/module"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// NewDirectoryModule is the factory method to create a new directory server module.
// On start this opens or creates a directory store in root/moduleID.
// Directory entries are stored in the 'directory' bucket.
//
// If a http server is provided this registers the HTTP API with the router and serves
// its TD on the .well-known/wot endpoint as per discovery specification.
//
// storageRoot is the root dir of the storage area. Use "" for testing with an in-memory store.
// router is the html server router to register the html API handlers with. nil to ignore.
func NewDirectoryModule(storageRoot string, httpServer transports.IHttpServer) directoryapi.IDirectoryServer {
	m := module.NewDirectoryModule(storageRoot, httpServer)
	return m
}
