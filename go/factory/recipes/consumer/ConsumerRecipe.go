package consumerrecipe

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/directory"
	directoryclient "github.com/hiveot/hivekit/go/cells/directory/client"
	"github.com/hiveot/hivekit/go/cells/router"
	router_service "github.com/hiveot/hivekit/go/cells/router/service"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	discovery_client "github.com/hiveot/hivekit/go/cells/transport/discovery/client"
	"github.com/hiveot/hivekit/go/cells/vcache"
	vcache_service "github.com/hiveot/hivekit/go/cells/vcache/service"
	"github.com/hiveot/hivekit/go/factory/recipes"
	factory_service "github.com/hiveot/hivekit/go/factory/service"
)

const valueCacheSlotName = "vcache-slot"

// ConsumerRecipeChain defines the cells for IoT consumers in order of instantiation.
var ConsumerRecipeChain = []api.CellDefinition{
	{
		// optional value cache slot
		Type: valueCacheSlotName,
	},
	{
		// use a directory client to read thing TDs
		Type:        directory.DirectoryClientCellType,
		Constructor: directoryclient.StartDirectoryClientFactory,
	},
	{
		// discover the server using DNS-SD
		// app can retrieve it with f.GetCell(discovery.DiscoveryClientCellType)
		Type:        discovery.DiscoveryClientCellType,
		Constructor: discovery_client.StartDiscoveryClientFactory,
	},
	{
		// the router manages client connections
		Type:        router.RouterCellType,
		Constructor: router_service.StartRouterServiceFactory,
		// TODO: add configuration for using auto-reconnect
		// TODO: add configuration for providing credentials
	},
}

// StartConsumerRecipe starts a recipe for general consumers.
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
func StartConsumerRecipe(
	f api.ICellFactory, withValueCache bool) (api.IRecipe, error) {

	// copy the chain
	chain := ConsumerRecipeChain[:]

	// support slots by inserting into the defined chain before starting the formation, not after.
	if withValueCache {
		modDef := api.CellDefinition{
			Type:        vcache.ValueCacheCellType,
			Constructor: vcache_service.StartValueCacheServiceFactory,
		}
		recipes.SetSlot(chain, valueCacheSlotName, modDef)
	}

	// linkto doesnt apply to a consumer chain
	r, err := factory_service.StartChainFormation(f, chain, nil)
	return r, err
}
