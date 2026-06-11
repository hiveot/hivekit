package tlsserverpkg

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/tlsserver"
	"github.com/hiveot/hivekit/go/modules/transport/tlsserver/internal"
)

// Create a new TLS server instance with the given configuration
func NewTLSServer(cfg *tlsserver.TLSServerConfig, authenticator transport.IAuthenticator) transport.IHttpServer {
	srv := internal.NewTLSServer(cfg, authenticator)
	return srv
}

// Create a new transport server instance for the provided factory environment
func NewTLSServerFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {

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
	cfg := tlsserver.NewTLSServerConfig(addr, env.HttpsPort, serverCert, caCert, true)
	srv := internal.NewTLSServer(cfg, f.GetAuthenticator())

	return srv, nil
}
