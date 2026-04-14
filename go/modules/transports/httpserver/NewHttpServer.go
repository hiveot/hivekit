package httpserver

import (
	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	httpserverapi "github.com/hiveot/hivekit/go/modules/transports/httpserver/api"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver/internal"
)

// Create a new http server instance with the given configuration
func NewHttpServerModule(cfg *httpserverapi.Config) transports.IHttpServer {
	srv := internal.NewHttpServerModule(cfg)
	return srv
}

// Create a new http server instance for the provided factory environment
func NewHttpServerFactory(f factoryapi.IModuleFactory) modules.IHiveModule {

	env := f.GetEnvironment()
	caCert, err := env.GetCA()
	if err != nil {
		panic("unable to get the CA")
	}
	serverCert, err := env.GetServerCert()
	if err != nil {
		panic("unable to get the Server certificate")
	}
	authn := f.GetAuthenticator()
	addr := ""
	port := transports.DefaultHttpsPort
	cfg := httpserverapi.NewConfig(addr, port, serverCert, caCert, authn)

	srv := internal.NewHttpServerModule(cfg)
	return srv
}
