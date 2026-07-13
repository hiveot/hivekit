package routerpkg

import (
	"crypto/x509"
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/router"
	"github.com/hiveot/hivekit/go/modules/router/internal"
)

// NewRouterService creates a new instance of the router service module with the default module ID.
// Start must be called before usage.
//
//	storageDir location where the module stores its data
//	getTD  handler to lookup a TD for a thingID from a directory
//	getSrv handler to return the running list of transport servers that can contain
//	 reverse connections. nil to not support RCs.
//	caCert is the CA certificate used to verify device connections
//	timeout is the maximum wait time for sending requests to clients.
func NewRouterService(storageDir string,
	getTD func(thingID string) *td.TD,
	getSrv func() []api.ITransportServer,
	caCert *x509.Certificate, timeout time.Duration,
) router.IRouterService {

	m := internal.NewRouterServiceImpl(storageDir, getTD, getSrv, caCert, timeout)
	return m
}

// Create a router service instance using the factory environment
// This loads the directory module to lookup a Thing TD
func NewRouterServiceFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	var getTD func(string) *td.TD
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(router.RouterModuleType)

	// The router can also be used server and client side. Check for both server and client directory.
	m, err := f.StartModule(directory.DirectoryServiceModuleType, true)
	if err == nil {
		if dirMod, ok := m.(directory.IDirectoryService); ok {
			getTD = dirMod.GetTD
		}
	} else {
		// maybe directory client?
		m, err = f.StartModule(directory.DirectoryClientModuleType, true)
		if err == nil {
			if dirMod, ok := m.(directory.IDirectoryClient); ok {
				getTD = dirMod.Cache().GetThing
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("NewRouterServiceFactory. Missing TD directory: %w", err)
	}
	svc := NewRouterService(storageDir, getTD, f.GetTransportServers, env.CaCert, f.GetEnvironment().RpcTimeout)
	svc.SetTimeout(env.RpcTimeout)

	return svc, nil
}
