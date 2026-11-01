package tlsserver

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
)

// Create a ready-to-use TLS server instance with the given configuration.
func NewTLSServer(cfg *TLSServerConfig, authenticator api.IAuthenticator) (
	api.IHttpServer, error) {

	return NewTLSServerImpl(cfg, authenticator)
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
	showLogging := env.LogLevel == "info" || env.LogLevel == "debug"
	cfg := NewTLSServerConfig(
		addr, env.HttpsPort, serverCert, rootCAs, showLogging)
	return NewTLSServerImpl(cfg, f.GetAuthenticator())
}
