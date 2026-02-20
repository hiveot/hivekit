package pipeline

import (
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	directory_module "github.com/hiveot/hivekit/go/modules/directory/module"
	"github.com/hiveot/hivekit/go/modules/transports"
)

const DirectoryClient = "directoryClient"

// ModuleFactory for creating instances of pipeline modules for the pipeline environment
// the pipeline environment includes a storage area, optional http router and certificates
// the clientID and authToken are used when instantiating client interfaces of modules.
type ModuleFactory struct {
	caCert *x509.Certificate
	// the server certification for transport modules
	serverCert *tls.Certificate
	// when connecting a client interface using NewModuleClient
	clientID string
	// authentication token ...?
	authToken string
	// the root directory of the configuration storage directory
	configRoot string
	// the root directory of the application storage area (subdir per module)
	storageRoot string
	// the default timeout for transport modules
	timeout time.Duration
	// the http server with and modules that serve http endpoints
	httpServer transports.IHttpServer
}

// NewModule returns a new instance of a module with the given name.
// Start() must be called before usage.
// If the name is unknown this returns nil.
func (f *ModuleFactory) NewModule(name string) (m modules.IHiveModule) {

	switch name {
	case DirectoryClient:
		m = directory_module.NewDirectoryModule(f.storageRoot, f.httpServer)

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
		clientID:   clientID,
		authToken:  token,
		caCert:     caCert,
		timeout:    timeout,
		httpServer: nil,
	}
	return &f
}
