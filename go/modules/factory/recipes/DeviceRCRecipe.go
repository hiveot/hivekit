package recipes

import (
	"github.com/hiveot/hivekit/go/modules/certs"
	certspkg "github.com/hiveot/hivekit/go/modules/certs/pkg"
	"github.com/hiveot/hivekit/go/modules/factory"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/reconnect"
	reconnectpkg "github.com/hiveot/hivekit/go/modules/reconnect/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/clients"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transport/discovery/pkg"
)

// RCDeviceChain defines the module chain for use by IoT devices that use reverse
// connection to a gateway or hub.
// The IoT device logic can be added at the end using AppendModule or linking to it.
var RCDeviceChain = []factory.ModuleDefinition{
	{
		// initialize client certs / auth token in app environment
		Type:        certs.InitFactoryCertsModuleType,
		Constructor: certspkg.NewInitFactoryCerts,
	},
	{
		// discover the server using DNS-SD
		Type:        discovery.DiscoveryClientModuleType,
		Constructor: discoverypkg.NewDiscoveryClientFactory,
	},
	{
		// enable auto-reconnect for the client
		Type:        reconnect.ReconnectModuleType,
		Constructor: reconnectpkg.NewReconnectFactory,
	},
	{
		// connect a new client to the discovered server
		Type:        clients.TransportClientModuleType,
		Constructor: clients.NewTransportClientFactory,
	},
	// todo: add optional logging of requests
	// todo: optional authorization of requests

	// add and link your application module
}

// RCDeviceRecipe is a recipe for creating a reverse-connected devices.
// Intended for IoT devices that use reverse connection to a gateway or Hub.
//
// * support AppEnvironment commandline options
// * load CA and client certificate, and auth token if found
// * auto-discovery gateway/hub server URL if not provided
// * use gateway TD if available, fallback to serverURL scheme for protocol
// * enable auto-reconnect
// * establish client connection
//
// f is the module factory to use to use.
// appModule is the module definition of the exposed thing to inject in the app slot.
//
// This returns the recipe, which can be used like any other module
func NewRCDeviceRecipe(
	f factory.IModuleFactory, appModule *factory.ModuleDefinition) factory.IRecipe {
	chain := RCDeviceChain
	if appModule != nil {
		chain = append(chain, *appModule)
	}
	r := factorypkg.NewChainRecipe(f, chain)
	return r
}
