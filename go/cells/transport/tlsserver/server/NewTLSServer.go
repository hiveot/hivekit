package tls_server

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/transport/tlsserver"
	"github.com/hiveot/hivekit/go/cells/transport/tlsserver/internal"
)

// Create a ready-to-use TLS server instance with the given configuration.
func NewTLSServer(
	cfg *tlsserver.TLSServerConfig, authenticator api.IAuthenticator) (api.IHttpServer, error) {
	return internal.NewTLSServerImpl(cfg, authenticator)
}

// Create a ready-to-use TLS transport server instance for the provided
// factory environment. This uses the environmnet https port, server certificate
// and root CAs.
func NewTLSServerFactory(
	f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	env := f.GetEnvironment()

	serverCert, err := env.GetServerCert()
	if err != nil {
		slog.Error("unable to get the Server certificate")
	}
	addr := ""
	rootCAs := env.GetRootCAs()
	cfg := tlsserver.NewTLSServerConfig(
		addr, env.HttpsPort, serverCert, rootCAs, true)
	return internal.NewTLSServerImpl(cfg, f.GetAuthenticator())
}
