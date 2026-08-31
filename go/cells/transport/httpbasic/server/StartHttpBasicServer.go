package httpbasic_server

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/transport/httpbasic"
	"github.com/hiveot/hivekit/go/cells/transport/httpbasic/internal/serverimpl"
)

// StartHttpBasicServer starts a new WoT server supporting the http-basic protocol
func StartHttpBasicServer(httpServer api.IHttpServer) (httpbasic.IHttpBasicServer, error) {
	return serverimpl.StartHttpBasicServerImpl(httpServer)
}

// Start a new instance of the HTTP-Basic server using the factory environment.
// This loads the httpserver transport and starts listening.
func StartHttpBasicServerFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	httpServer := f.GetHttpServer(true)
	if httpServer == nil {
		return nil, fmt.Errorf("StartHttpBasicServerFactory: Missing Http server")
	}
	return StartHttpBasicServer(httpServer)
}
