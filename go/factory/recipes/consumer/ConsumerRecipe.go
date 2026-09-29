package consumer_recipe

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/cells/directory"
	directoryclient "github.com/hiveot/hivekit/go/cells/directory/client"
	"github.com/hiveot/hivekit/go/cells/router"
	router_service "github.com/hiveot/hivekit/go/cells/router/service"
	"github.com/hiveot/hivekit/go/cells/transport/clients"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	discovery_client "github.com/hiveot/hivekit/go/cells/transport/discovery/client"
	"github.com/hiveot/hivekit/go/cells/vcache"
	vcache_service "github.com/hiveot/hivekit/go/cells/vcache/service"
	"github.com/hiveot/hivekit/go/factory/recipes"
	factory_service "github.com/hiveot/hivekit/go/factory/service"
)

const valueCacheSlotName = "vcache-slot"

// Recipe use-cases:
// a. consumer->device using env.ServerURL, direct connection, no TD
//			only useful if the affordances are known, or device is a gateway
//     		router bypasses TD lookup and creates client for URL
// b. consumer->Thing, using serverTD and router
//          Intended for connecting to stand-alone things with a single TD
//			TD is provided OOB or in router getTD callback.
//          router creates client connection using TD forms
// c. consumer->device, using DNS-SD discovery on ThingID or instanceName + router
//          Intended for connecting to stand-alone things by thingiD
//          1. discovery downloads TD
//          2. router creates client connection using TD forms
// d. consumer->device using directory client and ThingID
//          1a. directory client is stand-alone with OOB cache
//          1b. directory client uses env.DirTD (by discovery) to connect to the directory server
//          1c. directory client uses env.DirURL to locate the directory server
//          1d. directory client uses discovery to locate the directory server
//          2. directory client looks up TD
//          3. router creates connection using TD forms
// e. consumer->gateway using serverTD;
//			FIXME: this doesnt work yet
//                       dirURL arg or discovery (OK)
// consumer->gateway, find device in gateway, including gateway

// ConsumerRecipeChain defines the cells for IoT consumers in order of instantiation.
var ConsumerRecipeChain = []api.CellDefinition{
	{
		// optional value cache slot
		Type: valueCacheSlotName,
	},
	{
		// discover the directory or server using DNS-SD
		// app can retrieve it with f.GetCell(discovery.DiscoveryClientCellType)
		Type:        discovery.DiscoveryClientCellType,
		Constructor: discovery_client.NewDiscoveryClientFactory,
	},
	{
		// use a directory client to read thing TDs
		Type:        directory.DirectoryClientCellType,
		Constructor: directoryclient.NewDirectoryClientFactory,
	},
	{
		// If a gateway or server URL is provided then connect directly to it.
		// If no gateway/server URL is provided this cell is ignored.
		Type:        clients.TransportClientCellType,
		Constructor: clients.NewTransportClientFactory,
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

// Start the recipe and factory.
func (r *ConsumerRecipe) Start() {
	r.f.Start()
}

// Stop the recipe and factory.
func (r *ConsumerRecipe) Stop() {
	r.f.Stop()
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
// Invoke Start on the recipe to run the consumer.
//
// A value cache can be included to capture property updates and event notifications.
//
// This:
// * support HiveEnvironment commandline options
// * load CA and client certificate, and auth token if found
// * directory client for access to discovered devices
// * discovery client for locating devices and directories
// * router for connecting to clients
//
//	withValueCache set to include a value cache in the cell chain
//
// This returns the consumer recipe, which can be used as a consumer for applications.
func NewConsumerRecipe(env *api.HiveEnvironment, withValueCache bool) (
	r *ConsumerRecipe, f api.ICellFactory, err error) {

	f = factory_service.NewCellFactory(env, nil)

	// copy the chain
	// support slots by inserting into the defined chain before starting the formation, not after.
	chain := ConsumerRecipeChain[:]
	if withValueCache {
		modDef := api.CellDefinition{
			Type:        vcache.ValueCacheCellType,
			Constructor: vcache_service.NewValueCacheServiceFactory,
		}
		recipes.SetSlot(chain, valueCacheSlotName, modDef)
	}

	formation, err := factory_service.NewChainFormation(f, chain, nil)
	co := consumer.NewConsumer(formation, nil)

	r = &ConsumerRecipe{
		Consumer:  co,
		f:         f,
		formation: formation,
	}

	var _ api.IRecipe = r
	var _ *consumer.Consumer = r.Consumer // interface checks
	return r, f, err
}
