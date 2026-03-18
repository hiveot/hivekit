package httpserver

import (
	"github.com/hiveot/hivekit/go/modules/transports"
	httpserverapi "github.com/hiveot/hivekit/go/modules/transports/httpserver/api"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver/internal"
)

func NewHttpServerModule(cfg *httpserverapi.Config) transports.IHttpServer {
	srv := internal.NewHttpServerModule(cfg)
	return srv
}
