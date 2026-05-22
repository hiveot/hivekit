package directorypkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	internal "github.com/hiveot/hivekit/go/modules/directory/internal/httpserver"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
)

func NewDirectoryHttpServer(httpServer transports.IHttpServer) directory.IDirectoryHttpServer {
	m := internal.StartDirectoryHttpServer(httpServer)
	return m
}

// Deprecated: use http-basic interface for http interaction with the directory.
// factory for the directory http interface module
func NewDirectoryHttpServerFactoryDeprecated(f factory.IModuleFactory) modules.IHiveModule {

	httpServer := f.GetHttpServer(true)
	m := NewDirectoryHttpServer(httpServer)
	return m
}
