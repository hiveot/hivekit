package httpbasic_server

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/transport/httpbasic"
	"github.com/hiveot/hivekit/go/cells/transport/httpbasic/internal/serverimpl"
)

// NewHttpBasicServer creates a new WoT server supporting the http-basic protocol
func NewHttpBasicServer(httpServer api.IHttpServer) httpbasic.IHttpBasicServer {
	srv := serverimpl.NewHttpBasicServerImpl(httpServer)
	return srv
}

// Create a new instance of the HTTP-Basic server using the factory environment
// This loads the httpserver transport
func NewHttpBasicServerFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	httpServer := f.GetHttpServer(true)
	if httpServer == nil {
		return nil, fmt.Errorf("NewHttpBasicServerFactory: Missing Http server")
	}
	return NewHttpBasicServer(httpServer), nil
}
