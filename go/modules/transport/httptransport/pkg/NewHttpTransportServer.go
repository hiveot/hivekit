package httptransportpkg

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/httptransport"
	internal "github.com/hiveot/hivekit/go/modules/transport/httptransport/internal/server"
)

// Create a new TLS server instance with the given configuration
func NewHttpTransportServer(cfg *httptransport.Config, authenticator transport.IAuthenticator) transport.IHttpServer {
	srv := internal.NewHttpTransportServer(cfg, authenticator)
	return srv
}

// Create a new transport server instance for the provided factory environment
func NewHttpTransportServerFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {

	env := f.GetEnvironment()
	caCert, err := env.GetCA()
	if err != nil {
		slog.Error("unable to get the CA")
	}
	serverCert, err := env.GetServerCert()
	if err != nil {
		slog.Error("unable to get the Server certificate")
	}
	addr := ""
	cfg := httptransport.NewConfig(addr, env.HttpsPort, serverCert, caCert, true)
	srv := internal.NewHttpTransportServer(cfg, f.GetAuthenticator())

	return srv, nil
}
