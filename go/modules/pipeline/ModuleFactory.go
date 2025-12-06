package pipeline

import (
	"crypto/x509"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/client/directoryclient"
)

const DirectoryClient = "directoryClient"

// ModuleFactory for creating instances of pipeline modules
type ModuleFactory struct {
	clientID  string
	authToken string
	caCert    *x509.Certificate
	timeout   time.Duration
}

// NewModule returns a new instance of a module with the given name.
// If the name is unknown this returns nil.
func (f *ModuleFactory) NewModule(name string) (m modules.IHiveModule) {

	switch name {
	case DirectoryClient:
		m = directoryclient.NewDirectoryClient(f.clientID, f.authToken, f.caCert, f.timeout)
	}
	return m
}

// Create a new instance of the module factory
//
// clientID is the client ID the modules created with this factory identify as.
// token is the auth token for use by the modules
// caCert is the CA used for clients and servers
// timeout is the connection timeout for use by clients
func NewModuleFactory(clientID string, token string, caCert *x509.Certificate, timeout time.Duration) *ModuleFactory {
	f := ModuleFactory{
		clientID:  clientID,
		authToken: token,
		caCert:    caCert,
		timeout:   timeout,
	}
	return &f
}
