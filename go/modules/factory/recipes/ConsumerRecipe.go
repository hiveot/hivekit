package recipes

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/certs"
	certspkg "github.com/hiveot/hivekit/go/modules/certs/pkg"
	"github.com/hiveot/hivekit/go/modules/directory"
	directorypkg "github.com/hiveot/hivekit/go/modules/directory/pkg"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/router"
	routerpkg "github.com/hiveot/hivekit/go/modules/router/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transport/discovery/pkg"
)

// ConsumerRecipeChain defines the modules for IoT consumers in order of instantiation
// Link a consumer to this chain.
var ConsumerRecipeChain = []api.ModuleDefinition{
	{
		// initialize client certs / auth token in app environment
		Type:        certs.InitFactoryCertsModuleType,
		Constructor: certspkg.NewInitFactoryCerts,
	},
	// {
	// 	// application slot
	// 	Type: AppSlotType,
	// },
	{
		// use a directory client to read things
		Type:        directory.DirectoryClientModuleType,
		Constructor: directorypkg.NewDirectoryClientFactory,
	},
	{
		// discover the server using DNS-SD
		// app can retrieve it with f.GetModule(discovery.DiscoveryClientModuleType)
		Type:        discovery.DiscoveryClientModuleType,
		Constructor: discoverypkg.NewDiscoveryClientFactory,
	},
	{
		// the router manages client connections
		Type:        router.RouterModuleType,
		Constructor: routerpkg.NewRouterServiceFactory,
	},
}

// ConsumerRecipe.go is a recipe for general consumers
//
// This:
// * support AppEnvironment commandline options
// * load CA and client certificate, and auth token if found
// * directory client for access to discovered devices
// * discovery client for locating devices and directories
// * router for connecting to clients
//
// f is the module factory to use to use.
//
// This returns the recipe, which can be used as a module sink to a consumer module.
func NewConsumerRecipe(f api.IModuleFactory) api.IRecipe {

	chain := ConsumerRecipeChain

	r := factorypkg.NewChainRecipe(f, chain)
	return r
}
