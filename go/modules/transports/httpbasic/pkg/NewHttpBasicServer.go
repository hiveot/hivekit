package httpbasicpkg

import (
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic"
	internalserver "github.com/hiveot/hivekit/go/modules/transports/httpbasic/internal/server"
)

// NewHttpBasicServer creates a new WoT server supporting the http-basic protocol
func NewHttpBasicServer(httpServer transports.IHttpServer) httpbasic.IHttpBasicServer {
	srv := internalserver.NewHttpBasicServer(httpServer)
	return srv
}
