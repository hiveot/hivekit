package ssescpkg

import (
	"github.com/hiveot/hivekit/go/modules"
	factory "github.com/hiveot/hivekit/go/modules/factory"
)

// Create a new instance of the Hiveot SSE-SC server using the factory environment
// This loads the httpserver module
func NewSseScServerFactory(f factory.IModuleFactory) modules.IHiveModule {
	httpServer := f.GetHttpServer(true)
	timeout := f.GetEnvironment().RpcTimeout
	return NewSseScServer(httpServer, timeout)
}
