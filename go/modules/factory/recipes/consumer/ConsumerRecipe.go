package consumerrecipe

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/directory"
	directoryclient "github.com/hiveot/hivekit/go/modules/directory/client"
	factory_service "github.com/hiveot/hivekit/go/modules/factory/service"
	"github.com/hiveot/hivekit/go/modules/router"
	router_service "github.com/hiveot/hivekit/go/modules/router/service"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	discovery_client "github.com/hiveot/hivekit/go/modules/transport/discovery/client"
	"github.com/hiveot/hivekit/go/modules/vcache"
	vcache_service "github.com/hiveot/hivekit/go/modules/vcache/service"
)

const valueCacheSlot = "vcache-slot"

// ConsumerRecipeChain defines the modules for IoT consumers in order of instantiation.
var ConsumerRecipeChain = []api.ModuleDefinition{
	{
		// optional value cache slot
		Type: valueCacheSlot,
	},
	{
		// use a directory client to read thing TDs
		Type:        directory.DirectoryClientModuleType,
		Constructor: directoryclient.NewDirectoryClientFactory,
	},
	{
		// discover the server using DNS-SD
		// app can retrieve it with f.GetModule(discovery.DiscoveryClientModuleType)
		Type:        discovery.DiscoveryClientModuleType,
		Constructor: discovery_client.NewDiscoveryClientFactory,
	},
	{
		// the router manages client connections
		// FIXME: where does the router gets its client connection credentials from?
		Type:        router.RouterModuleType,
		Constructor: router_service.NewRouterServiceFactory,
	},
}

// ConsumerRecipe.go is a recipe for general consumers.
//
// A value cache can be included to capture property updates and event notifications.
//
// This:
// * support AppEnvironment commandline options
// * load CA and client certificate, and auth token if found
// * directory client for access to discovered devices
// * discovery client for locating devices and directories
// * router for connecting to clients
//
// f is the module factory to use to use.
// withValueCache set to include a value cache in the module chain
//
// This returns the recipe, which can be used as a module sink to a consumer module.
func NewConsumerRecipe(f api.IModuleFactory, withValueCache bool) api.IRecipe {

	chain := ConsumerRecipeChain

	r := factory_service.NewChainRecipe(f, chain)
	if withValueCache {
		modDef := api.ModuleDefinition{
			Type:        vcache.ValueCacheModuleType,
			Constructor: vcache_service.NewValueCacheServiceFactory,
		}
		r.SetSlot(valueCacheSlot, modDef)
	}
	return r
}
