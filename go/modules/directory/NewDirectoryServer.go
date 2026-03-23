package directory

import (
	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
	"github.com/hiveot/hivekit/go/modules/directory/internal"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// NewDirectoryService is the factory method to create a new Thing directory service module.
// On start this opens or creates a directory store in root/moduleID.
// Directory entries are stored in the 'directory' bucket.
//
// If a http server is provided this registers the HTTP API with the router and serves
// its TD on the .well-known/wot endpoint as per discovery specification.
//
// storageDir is the location where the module stores its data. Use "" for testing with an in-memory store.
// router is the html server router to register the html API handlers with. nil to ignore.
func NewDirectoryService(storageDir string, httpServer transports.IHttpServer) directoryapi.IDirectoryServer {
	m := internal.NewDirectoryService(storageDir, httpServer)
	return m
}
