package consumerrecipe

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/consumer"
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

// A consumer recipe is a consumer that includes directory and discovery clients
// with a router to connect to devices and gateways.
type ConsumerRecipe struct {
	*consumer.Consumer
	f         api.ICellFactory
	formation api.IRecipe
}

// Return the discovery client from this recipe
func (r *ConsumerRecipe) GetDiscovery() discovery.IDiscoveryClient {
	discoClient := api.GetFactoryCell[discovery.IDiscoveryClient](r.f, discovery.DiscoveryClientCellType)
	return discoClient
}

// Return the directory client from this recipe
func (r *ConsumerRecipe) GetDirectory() directory.IDirectoryClient {
	dirClient := api.GetFactoryCell[directory.IDirectoryClient](r.f, directory.DirectoryClientCellType)
	return dirClient
}

// Set the credentials to use for connecting to devices/services
// This updates the router with the credentials for the given deviceID.
//
// deviceID is the thingID of the device connecting to.
// In case of a gateway this is the gateway server thingID.
//
// This is useful for setting device specific credentials.
func (r *ConsumerRecipe) SetCredentials(
	deviceID string, clientID string, token string) {

	rtr := api.GetFactoryCell[router.IRouterService](r.f, router.RouterCellType)
	if token != "" {
		rtr.AddCredentials(deviceID, clientID, token, td.SecSchemeBearer)
	}
}

// NewConsumerRecipe returns a ready to use recipe usable as a consumer.
//
// The resulting consumer is the start of the recipe chain.
// Invoke Start on the provided factory to run the application.
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
// This returns the consumer recipe, which can be used as a consumer for applications.
func NewConsumerRecipe(f api.ICellFactory, withValueCache bool) (*ConsumerRecipe, error) {

	// copy the chain
	chain := ConsumerRecipeChain[:]

	// support slots by inserting into the defined chain before starting the formation, not after.
	if withValueCache {
		modDef := api.CellDefinition{
			Type:        vcache.ValueCacheCellType,
			Constructor: vcache_service.NewValueCacheServiceFactory,
		}
		recipes.SetSlot(chain, valueCacheSlotName, modDef)
	}

	formation, err := factory_service.NewChainFormation(f, chain, nil)
	co := consumer.NewConsumer(formation, nil)

	r := &ConsumerRecipe{
		Consumer:  co,
		f:         f,
		formation: formation,
	}

	var _ api.IRecipe = r
	var _ *consumer.Consumer = r.Consumer // interface checks
	return r, err
}
