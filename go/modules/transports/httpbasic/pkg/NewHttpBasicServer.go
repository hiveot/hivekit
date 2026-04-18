package httpbasicpkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic/internal/server"
)

// NewHttpBasicServer creates a new WoT server supporting the http-basic protocol
func NewHttpBasicServer(httpServer transports.IHttpServer) httpbasic.IHttpBasicServer {
	srv := server.NewHttpBasicServer(httpServer)
	return srv
}

// Create a new instance of the HTTP-Basic server using the factory environment
// This loads the httpserver module
func NewHttpBasicServerFactory(f factory.IModuleFactory) modules.IHiveModule {
	httpServer := f.GetHttpServer()
	return NewHttpBasicServer(httpServer)
}
