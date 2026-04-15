package router

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/api/td"
	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules"
	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
	routerapi "github.com/hiveot/hivekit/go/modules/router/api"
	"github.com/hiveot/hivekit/go/modules/router/internal"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// NewRouterService creates a new instance of the router service module with the default module ID.
// Start must be called before usage.
//
//	storageDir location where the module stores its data
//	getTD is the handler to lookup a TD for a thingID from a directory
//	transports is a list of transport servers that can contain reverse agent connections.
//	caCert is the CA certificate used to verify device connections
func NewRouterService(
	storageDir string,
	getTD func(thingID string) *td.TD,
	tps []transports.ITransportServer,
	caCert *x509.Certificate,
) routerapi.IRouterService {

	m := internal.NewRouterService(storageDir, getTD, tps, caCert)
	return m
}

// Create a router service instance using the factory environment
// This loads the directory module to lookup a Thing TD
func NewRouterServiceFactory(f factoryapi.IModuleFactory) modules.IHiveModule {
	var getTD func(string) *td.TD
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(routerapi.RouterModuleType)

	// FIXME: what if a new transport server is started after the router is started?
	// option 1: provide a method to retrieve them when needed
	tps := f.GetTransportServers()

	m, err := f.GetModule(directoryapi.DirectoryModuleType)
	if err == nil {
		if dirMod, ok := m.(directoryapi.IDirectoryServer); ok {
			getTD = dirMod.GetTD
		}
	}
	return NewRouterService(storageDir, getTD, tps, env.CaCert)
}
