package directory_service

import (
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/directory"
	internal "github.com/hiveot/hivekit/go/modules/directory/internal/httpserver"
)

// Create a new instance
func NewDirectoryHttpServer(httpServer api.IHttpServer, respTimeout time.Duration) directory.IDirectoryHttpServer {
	m := internal.NewDirectoryHttpServer(httpServer, respTimeout)
	return m
}

// Factory for the directory http interface module
// Place this before the directory server module in the chain and before middleware modules that log and
// authorize requests.
func NewDirectoryHttpServerFactory(f api.IModuleFactory) api.IHiveModule {

	rpcTimeout := f.GetEnvironment().RpcTimeout
	httpServer, ok := f.GetModule(api.HttpServerModuleType).(api.IHttpServer)
	_ = ok
	m := internal.NewDirectoryHttpServer(httpServer, rpcTimeout)
	return m
}
