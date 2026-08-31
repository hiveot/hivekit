package rcdevicerecipe

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/reconnect"
	reconnect_service "github.com/hiveot/hivekit/go/cells/reconnect/service"
	"github.com/hiveot/hivekit/go/cells/transport/clients"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	discovery_client "github.com/hiveot/hivekit/go/cells/transport/discovery/client"
	factory_service "github.com/hiveot/hivekit/go/factory/service"
)

// Cell type name of the slot where to insert the 'exposed thing' application cell.
const AppSlotType = "appSlot"

// RCDeviceChain defines a chain for IoT devices that use reverse connection to a gateway or hub.
// The IoT device logic can be added at the end using AppendCell or linking to it.
var RCDeviceChain = []api.CellDefinition{
	{
		// discover the server running the directory
		// this sets the factory serverTD
		Type:        discovery.DiscoveryClientCellType,
		Constructor: discovery_client.StartDiscoveryClientFactory,
	},
	{
		// enable auto-reconnect for the client
		Type:        reconnect.ReconnectCellType,
		Constructor: reconnect_service.StartReconnectFactory,
	},
	{
		// connect a new client to the discovered server
		// the server TD is set by discovery.
		Type:        clients.TransportClientCellType,
		Constructor: clients.NewTransportClientFactory,
	},
	// todo: add optional logging of requests
	// todo: optional authorization of requests

	// add and link your application cell, which will handle requests
	// or use the app slot.
	{
		// Cell slot for the application cell.
		// This place lets it publish its TD for discovery as it is placed before those cells.
		// Use Chain.SetSlot(AppSlotType, cellDef)
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
//	f is the cell factory to use to use.
//	appCellDef is the cell definition of the exposed thing to inject in the app slot.
//
// This returns the recipe, which can be used like any other cell
func NewRCDeviceRecipe(
	f api.ICellFactory, appCellDef *api.CellDefinition) api.IRecipe {
	chain := RCDeviceChain
	if appCellDef != nil {
		chain = append(chain, *appCellDef)
	}
	r := factory_service.NewChainFormation(f, chain)
	return r
}
