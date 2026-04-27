package httptransportpkg

import (
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httptransport"
	internal "github.com/hiveot/hivekit/go/modules/transports/httptransport/internal/server"
)

// Create a new TLS server instance with the given configuration
func NewHttpTransportServer(cfg *httptransport.Config) transports.IHttpServer {
	srv := internal.NewHttpTransportServer(cfg)
	return srv
}
