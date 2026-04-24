package directorypkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	directoryhttp "github.com/hiveot/hivekit/go/modules/directory/internal/httpapi"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
)

func NewDirectoryHttpHandler(httpServer transports.IHttpServer) directory.IDirectoryHttpServer {
	m := directoryhttp.StartDirectoryHttpHandler(httpServer)
	return m
}

// factory for the directory http interface module
func NewDirectoryHttpHandlerFactory(f factory.IModuleFactory) modules.IHiveModule {

	httpServer := f.GetHttpServer(true)
	m := NewDirectoryHttpHandler(httpServer)
	return m
}
