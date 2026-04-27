package httptransportpkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httptransport"
	internal "github.com/hiveot/hivekit/go/modules/transports/httptransport/internal/server"
)

// Create a new transport server instance for the provided factory environment
func NewHttpTransportServerFactory(f factory.IModuleFactory) modules.IHiveModule {

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
	cfg := httptransport.NewConfig(addr, port, serverCert, caCert, authenticator, true)
	srv := internal.NewHttpTransportServer(cfg)
	return srv
}
