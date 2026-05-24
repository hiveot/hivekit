package directorypkg

import (
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	internal "github.com/hiveot/hivekit/go/modules/directory/internal/httpserver"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// Create a new instance
func NewDirectoryHttpServer(httpServer transports.IHttpServer, respTimeout time.Duration) directory.IDirectoryHttpServer {
	m := internal.StartDirectoryHttpServer(httpServer, respTimeout)
	return m
}

// Factory for the directory http interface module
// Place this before the directory server module in the chain and before middleware modules that log and
// authorize requests.
func NewDirectoryHttpServerFactory(f factory.IModuleFactory) modules.IHiveModule {

	httpServer := f.GetHttpServer(true)
	m := internal.StartDirectoryHttpServer(httpServer, f.GetEnvironment().RpcTimeout)
	return m
}
