package directory

import (
	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
	directoryserver "github.com/hiveot/hivekit/go/modules/directory/internal/server"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// NewDirectoryServer is the factory method to create a new directory server module.
// On start this opens or creates a directory store in root/moduleID.
// Directory entries are stored in the 'directory' bucket.
//
// If a http server is provided this registers the HTTP API with the router and serves
// its TD on the .well-known/wot endpoint as per discovery specification.
//
// storageRoot is the root dir of the storage area. Use "" for testing with an in-memory store.
// router is the html server router to register the html API handlers with. nil to ignore.
func NewDirectoryServer(storageRoot string, httpServer transports.IHttpServer) directoryapi.IDirectoryServer {
	m := directoryserver.NewDirectoryServer(storageRoot, httpServer)
	return m
}
