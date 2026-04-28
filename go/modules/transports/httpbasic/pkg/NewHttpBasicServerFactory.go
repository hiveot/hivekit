package httpbasicpkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
)

// Create a new instance of the HTTP-Basic server using the factory environment
// This loads the httpserver module
func NewHttpBasicServerFactory(f factory.IModuleFactory) modules.IHiveModule {
	httpServer := f.GetHttpServer(true)
	return NewHttpBasicServer(httpServer)
}
