package consumerrecipe

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/directory"
	directoryclient "github.com/hiveot/hivekit/go/cells/directory/client"
	factory_service "github.com/hiveot/hivekit/go/cells/factory/service"
	"github.com/hiveot/hivekit/go/cells/router"
	router_service "github.com/hiveot/hivekit/go/cells/router/service"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	discovery_client "github.com/hiveot/hivekit/go/cells/transport/discovery/client"
	"github.com/hiveot/hivekit/go/cells/vcache"
	vcache_service "github.com/hiveot/hivekit/go/cells/vcache/service"
)

const valueCacheSlot = "vcache-slot"

// ConsumerRecipeChain defines the cells for IoT consumers in order of instantiation.
var ConsumerRecipeChain = []api.CellDefinition{
	{
		// optional value cache slot
		Type: valueCacheSlot,
	},
	{
		// use a directory client to read thing TDs
		Type:        directory.DirectoryClientCellType,
		Constructor: directoryclient.NewDirectoryClientFactory,
	},
	{
		// discover the server using DNS-SD
		// app can retrieve it with f.GetCell(discovery.DiscoveryClientCellType)
		Type:        discovery.DiscoveryClientCellType,
		Constructor: discovery_client.NewDiscoveryClientFactory,
	},
	{
		// the router manages client connections
		Type:        router.RouterCellType,
		Constructor: router_service.NewRouterServiceFactory,
		// TODO: add configuration for using auto-reconnect
		// TODO: add configuration for providing credentials
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
//	f is the factory to use.
//	withValueCache set to include a value cache in the cell chain
//
// This returns the recipe, which can be used as a request sink to a consumer cell.
func NewConsumerRecipe(f api.ICellFactory, withValueCache bool) api.IRecipe {

	chain := ConsumerRecipeChain

	r := factory_service.NewChainRecipe(f, chain)
	if withValueCache {
		modDef := api.CellDefinition{
			Type:        vcache.ValueCacheCellType,
			Constructor: vcache_service.NewValueCacheServiceFactory,
		}
		r.SetSlot(valueCacheSlot, modDef)
	}
	return r
}
