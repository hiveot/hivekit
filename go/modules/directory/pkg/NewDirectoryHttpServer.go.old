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

// factory for the directory http interface module
func NewDirectoryHttpServerFactory(f factory.IModuleFactory) modules.IHiveModule {

	httpServer := f.GetHttpServer(true)
	m := NewDirectoryHttpServer(httpServer)
	return m
}
