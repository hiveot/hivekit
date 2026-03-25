package httpbasic

import (
	"github.com/hiveot/hivekit/go/modules/transports"
	httpbasicapi "github.com/hiveot/hivekit/go/modules/transports/httpbasic/api"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic/internal/server"
)

// NewHttpBasicServer creates a new WoT server supporting the http-basic protocol
func NewHttpBasicServer(httpServer transports.IHttpServer) httpbasicapi.IHttpBasicServer {
	srv := server.NewHttpBasicServer(httpServer)
	return srv
}
