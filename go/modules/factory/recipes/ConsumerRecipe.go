package recipes

import (
	"github.com/hiveot/hivekit/go/modules/certs"
	certspkg "github.com/hiveot/hivekit/go/modules/certs/pkg"
	"github.com/hiveot/hivekit/go/modules/factory"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/reconnect"
	reconnectpkg "github.com/hiveot/hivekit/go/modules/reconnect/pkg"
	"github.com/hiveot/hivekit/go/modules/router"
	routerpkg "github.com/hiveot/hivekit/go/modules/router/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transport/discovery/pkg"
)

// ConsumerRecipeChain defines the modules for IoT consumers in order of instantiation
var ConsumerRecipeChain = []factory.ModuleDefinition{
	{
		// initialize client certs / auth token in app environment
		Type:        certs.InitFactoryCertsModuleType,
		Constructor: certspkg.NewInitFactoryCerts,
	},
	{
		// application slot
		Type: AppSlotType,
	},
	{
		// discover the server using DNS-SD
		// app can retrieve it with f.GetModule(discovery.DiscoveryClientModuleType)
		Type:        discovery.DiscoveryClientModuleType,
		Constructor: discoverypkg.NewDiscoveryClientFactory,
	},
	{
		// enable auto-reconnect for the client on server restart
		Type:        reconnect.ReconnectModuleType,
		Constructor: reconnectpkg.NewReconnectFactory,
	},
	{
		// the router manages client connections
		Type:        router.RouterModuleType,
		Constructor: routerpkg.NewRouterServiceFactory,
	},
}

// ConsumerRecipe.go is a recipe for general consumers that do not use a gateway.
//
// This recipe places a consumer behind the application in the chain. The
// application can obtain it as its request sink.
//
// * support AppEnvironment commandline options
// * load CA and client certificate, and auth token if found
// * slot for applications
// * consumer for application.
// * enable auto-reconnect for possible client connections
//
// f is the module factory to use to use.
// appModule is the optional application module to prepend to the chain
//
// This returns the recipe, which can be used like any other module
func NewConsumerRecipe(
	f factory.IModuleFactory, appModule *factory.ModuleDefinition) factory.IRecipe {

	chain := ConsumerRecipeChain

	r := factorypkg.NewChainRecipe(f, chain)
	// place the application module before
	if appModule != nil {
		r.SetSlot(AppSlotType, *appModule)
	}
	return r
}
