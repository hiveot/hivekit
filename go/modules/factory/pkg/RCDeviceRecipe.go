package factorypkg

import (
	"github.com/hiveot/hivekit/go/modules/certs"
	certspkg "github.com/hiveot/hivekit/go/modules/certs/pkg"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/reconnect"
	reconnectpkg "github.com/hiveot/hivekit/go/modules/reconnect/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/clients"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transport/discovery/pkg"
)

// RCDeviceChain defines the modules for IoT device in the order to instantiate and link.
// The IoT device logic can be added at the end using AppendModule or linking to it.
var RCDeviceChain = []factory.ModuleDefinition{
	{
		Type:        certs.InitFactoryCertsModuleType,
		Constructor: certspkg.NewInitFactoryCerts,
	},
	{
		Type:        discovery.DiscoveryClientModuleType,
		Constructor: discoverypkg.NewDiscoveryClientFactory,
	},
	{
		Type:        reconnect.ReconnectModuleType,
		Constructor: reconnectpkg.NewReconnectFactory,
	},
	{
		Type:        clients.TransportClientModuleType,
		Constructor: clients.NewTransportClientFactory,
	},
}

// RCDeviceRecipe is a recipe template for creating a reverse-connection devices.
// Intended for IoT devices that use reverse connection to a gateway or Hub.
//
// * support AppEnvironment commandline options
// * auto-discovery gateway/hub server URL if not provided
// * use gateway TD if available, fallback to serverURL scheme for protocol
// * establish client connection
// * auto-reconnect
//
// env is the application environment to use
// appModule is the optional application module to append to the chain
//
// This returns the recipe, which can be used like any other module
func NewRCDeviceRecipe(
	f factory.IModuleFactory, appModule *factory.ModuleDefinition) *ChainRecipe {
	chain := RCDeviceChain
	if appModule != nil {
		chain = append(chain, *appModule)
	}
	r := NewChainRecipe(f, chain)
	return r
}
