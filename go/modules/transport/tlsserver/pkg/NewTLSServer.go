package tlsserverpkg

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/transport/tlsserver"
	"github.com/hiveot/hivekit/go/modules/transport/tlsserver/internal"
)

// Create a new TLS server instance with the given configuration
func NewTLSServer(cfg *tlsserver.TLSServerConfig, authenticator api.IAuthenticator) api.IHttpServer {
	srv := internal.NewTLSServer(cfg, authenticator)
	return srv
}

// Create a new http transport server instance for the provided factory environment.
// This uses the appp ID as the server and certificate name.
func NewTLSServerFactory(
	f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {

	env := f.GetEnvironment()

	caCert, err := env.GetCACert()
	if err != nil {
		slog.Error("unable to get the CA")
	}
	serverCert, err := env.GetTLSCert()
	if err != nil {
		slog.Error("unable to get the Server certificate")
	}
	addr := ""
	cfg := tlsserver.NewTLSServerConfig(addr, env.HttpsPort, serverCert, caCert, true)
	srv := internal.NewTLSServer(cfg, f.GetAuthenticator())
	return srv, nil
}
