package httpserver

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
	httpserverconfig "github.com/hiveot/hivekit/go/modules/transports/httpserver/config"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver/internal"
)

// Create a new TLS server instance with the given configuration
func NewHttpServerModule(cfg *httpserverconfig.Config) transports.IHttpServer {
	srv := internal.NewHttpServer(cfg)
	return srv
}

// Create a new TLS server instance for the provided factory environment
func NewHttpServerFactory(f factory.IModuleFactory) modules.IHiveModule {

	env := f.GetEnvironment()
	caCert, err := env.GetCA()
	if err != nil {
		panic("unable to get the CA")
	}
	serverCert, err := env.GetServerCert()
	if err != nil {
		panic("unable to get the Server certificate")
	}
	authenticator := f.GetAuthenticator()
	addr := ""
	port := transports.DefaultHttpsPort
	cfg := httpserverconfig.NewConfig(addr, port, serverCert, caCert, authenticator, true)

	srv := internal.NewHttpServer(cfg)
	return srv
}
