package httpbasic

import (
	"github.com/hiveot/hivekit/go/modules/transports"
	httpbasicapi "github.com/hiveot/hivekit/go/modules/transports/httpbasic/api"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic/internal"
)

func NewHttpBasicTransport(httpServer transports.IHttpServer) httpbasicapi.IHttpBasicTransport {
	srv := internal.NewHttpBasicTransport(httpServer)
	return srv
}
