package httpbasic_server

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/transport/httpbasic"
	"github.com/hiveot/hivekit/go/cells/transport/httpbasic/internal/serverimpl"
)

// NewHttpBasicServer returns a ready-to-use  WoT server supporting the http-basic protocol
func NewHttpBasicServer(httpServer api.IHttpServer) (httpbasic.IHttpBasicServer, error) {
	return serverimpl.NewHttpBasicServerImpl(httpServer)
}

// NewHttpBasicServerFactory returns a ready-to-use HTTP-Basic server using the
// http server from the factory environment.
// This loads the httpserver transport and starts listening.
func NewHttpBasicServerFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	httpServer := f.GetHttpServer(true)
	if httpServer == nil {
		return nil, fmt.Errorf("StartHttpBasicServerFactory: Missing Http server")
	}
	return NewHttpBasicServer(httpServer)
}
