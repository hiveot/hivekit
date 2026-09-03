package tls_server

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/transport/tlsserver"
	"github.com/hiveot/hivekit/go/cells/transport/tlsserver/internal"
)

// Create a new TLS server instance with the given configuration
func StartTLSServer(
	cfg *tlsserver.TLSServerConfig, authenticator api.IAuthenticator) (api.IHttpServer, error) {
	return internal.StartTLSServerImpl(cfg, authenticator)
}

// Create a new http transport server instance for the provided factory environment.
// This uses the appp ID as the server and certificate name.
func StartTLSServerFactory(
	f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	env := f.GetEnvironment()

	serverCert, err := env.GetServerCert()
	if err != nil {
		slog.Error("unable to get the Server certificate")
	}
	addr := ""
	rootCAs := env.GetRootCAs()
	cfg := tlsserver.NewTLSServerConfig(addr, env.HttpsPort, serverCert, rootCAs, true)
	return internal.StartTLSServerImpl(cfg, f.GetAuthenticator())
}
