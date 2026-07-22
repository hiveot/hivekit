package rcdevicerecipe

import (
	"github.com/hiveot/hivekit/go/api"
	factory_service "github.com/hiveot/hivekit/go/modules/factory/service"
	"github.com/hiveot/hivekit/go/modules/reconnect"
	reconnect_service "github.com/hiveot/hivekit/go/modules/reconnect/service"
	"github.com/hiveot/hivekit/go/modules/transport/clients"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	discovery_client "github.com/hiveot/hivekit/go/modules/transport/discovery/client"
)

// module type name of the slot where to insert the 'exposed thing' application module.
const AppSlotType = "appSlot"

// RCDeviceChain defines a client module chain for IoT devices that use reverse
// connection to a gateway or hub.
// The IoT device logic can be added at the end using AppendModule or linking to it.
var RCDeviceChain = []api.ModuleDefinition{
	{
		// discover the server running the directory
		Type:        discovery.DiscoveryClientModuleType,
		Constructor: discovery_client.NewDiscoveryClientFactory,
	},
	{
		// enable auto-reconnect for the client
		Type:        reconnect.ReconnectModuleType,
		Constructor: reconnect_service.NewReconnectFactory,
	},
	{
		// connect a new client to the discovered server
		// the server URL is set by discovery.
		Type:        clients.TransportClientModuleType,
		Constructor: clients.NewTransportClientFactory,
	},
	// todo: add optional logging of requests
	// todo: optional authorization of requests

	// add and link your application module, which will handle requests
	// or use the app slot.
	{
		// Module slot for the application module.
		// This is the application module. This place lets it publish its TD for discovery as it is
		// placed before those modules.
		// Use Chain.SetSlot(AppSlotType, moduleDef)
		Type: AppSlotType,
	},
	// Q: how does the device write its TD to the directory?
	// A: Use directorypkg.UpdateTD(dirThingID, tdjson, recipe-as-sink)
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
	f api.IModuleFactory, appModule *api.ModuleDefinition) api.IRecipe {
	chain := RCDeviceChain
	if appModule != nil {
		chain = append(chain, *appModule)
	}
	r := factory_service.NewChainRecipe(f, chain)
	return r
}
