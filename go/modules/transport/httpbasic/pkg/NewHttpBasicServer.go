package httpbasicpkg

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/transport/httpbasic"
	internalserver "github.com/hiveot/hivekit/go/modules/transport/httpbasic/internal/server"
)

// NewHttpBasicServer creates a new WoT server supporting the http-basic protocol
func NewHttpBasicServer(httpServer api.IHttpServer) httpbasic.IHttpBasicServer {
	srv := internalserver.NewHttpBasicServer(httpServer)
	return srv
}

// Create a new instance of the HTTP-Basic server using the factory environment
// This loads the httpserver module
func NewHttpBasicServerFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	httpServer := f.GetHttpServer(true)
	if httpServer == nil {
		return nil, fmt.Errorf("NewHttpBasicServerFactory: Missing Http server")
	}
	return NewHttpBasicServer(httpServer), nil
}
