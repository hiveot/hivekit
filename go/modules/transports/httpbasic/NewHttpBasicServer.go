package httpbasic

import (
	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	httpbasicapi "github.com/hiveot/hivekit/go/modules/transports/httpbasic/api"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic/internal/server"
)

// NewHttpBasicServer creates a new WoT server supporting the http-basic protocol
func NewHttpBasicServer(httpServer transports.IHttpServer) httpbasicapi.IHttpBasicServer {
	srv := server.NewHttpBasicServer(httpServer)
	return srv
}

// Create a new instance of the HTTP-Basic server using the factory environment
// This loads the httpserver module
func NewHttpBasicServerFactory(f factoryapi.IModuleFactory) modules.IHiveModule {
	httpServer := f.GetHttpServer()
	return NewHttpBasicServer(httpServer)
}
