package wsspkg

import (
	"github.com/hiveot/hivekit/go/modules"
	factory "github.com/hiveot/hivekit/go/modules/factory"
)

// Load the HiveOT websocket server using the factory environment
// This loads the http server
func NewHiveotWssServerFactory(f factory.IModuleFactory) modules.IHiveModule {
	httpServer := f.GetHttpServer(true)
	timeout := f.GetEnvironment().RpcTimeout
	return NewHiveotWssServer(httpServer, timeout)
}

// Load the Wot websocket server using the factory environment
// This loads the http server
func NewWotWssServerFactory(f factory.IModuleFactory) modules.IHiveModule {
	httpServer := f.GetHttpServer(true)
	timeout := f.GetEnvironment().RpcTimeout
	return NewWotWssServer(httpServer, timeout)
}
