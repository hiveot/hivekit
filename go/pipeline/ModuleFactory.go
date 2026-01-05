package pipeline

import (
	"crypto/x509"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/modules"
	directory_module "github.com/hiveot/hivekit/go/modules/services/directory/module"
)

const DirectoryClient = "directoryClient"

// ModuleFactory for creating instances of pipeline modules for the pipeline environment
// the pipeline environment includes a storage area, optional http router and certificates
// the clientID and authToken are used when instantiating client interfaces of modules.
type ModuleFactory struct {
	caCert *x509.Certificate
	// when connecting a client interface using NewModuleClient
	clientID  string
	authToken string
	// the root directory of the application storage area
	storageRoot string
	timeout     time.Duration
	// the server router is used to tie a http server with and modules that serve http endpoints
	serverHttpRouter *chi.Mux
}

// NewModule returns a new instance of a module with the given name.
// Start() must be called before usage.
// If the name is unknown this returns nil.
func (f *ModuleFactory) NewModule(name string) (m modules.IHiveModule) {

	switch name {
	case DirectoryClient:
		m = directory_module.NewDirectoryModule(f.storageRoot, f.serverHttpRouter)

	}
	return m
}

// NewModuleClient returns a client instance of a module with the given name.
// If the name is unknown this returns nil.
// func (f *ModuleFactory) NewModuleClient(name string) (m modules.IHiveModule) {

// 	switch name {
// 	case DirectoryClient:
// 		m = directory_api.NewDirectoryMsgClient("")
// 	}
// 	return m
// }

// Create a new instance of the module factory
//
// clientID is the client ID the modules created with this factory identify as.
// token is the auth token for use by the modules
// caCert is the CA used for clients and servers
// timeout is the connection timeout for use by clients
func NewModuleFactory(clientID string, token string, caCert *x509.Certificate, timeout time.Duration) *ModuleFactory {
	f := ModuleFactory{
		clientID:         clientID,
		authToken:        token,
		caCert:           caCert,
		timeout:          timeout,
		serverHttpRouter: chi.NewMux(),
	}
	return &f
}
