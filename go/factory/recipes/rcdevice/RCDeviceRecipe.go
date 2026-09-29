package rcdevice_recipe

import (
	"os"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/reconnect"
	reconnect_service "github.com/hiveot/hivekit/go/cells/reconnect/service"
	"github.com/hiveot/hivekit/go/cells/transport/clients"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	discovery_client "github.com/hiveot/hivekit/go/cells/transport/discovery/client"
	factory_service "github.com/hiveot/hivekit/go/factory/service"
)

// Cell type name of the slot where to insert the 'exposed thing' application cell.
const AppSlotType = "appSlot"

// RCDeviceChain defines a cell chain for IoT devices that use reverse connection
// to a gateway or hub.
// The cell with the IoT device logic is provided on start, which links it to the
// tail of the chain, where it will receive requests.
var RCDeviceChain = []api.CellDefinition{
	{
		// discover the server running the directory
		// this sets the factory serverTD
		Type:        discovery.DiscoveryClientCellType,
		Constructor: discovery_client.NewDiscoveryClientFactory,
	},
	{
		// enable auto-reconnect for the client
		Type:        reconnect.ReconnectCellType,
		Constructor: reconnect_service.NewReconnectServiceFactory,
	},
	{
		// connect a client to the gateway server.
		// the server TD is set by discovery, or the gateway URL is set manually.
		// This client will receive requests and pass it to the linked exposed thing
		//
		// For the exposed thing to publish its TD, it should send it to the
		// directory on the server.
		Type:        clients.TransportClientCellType,
		Constructor: clients.NewTransportClientFactory,
	},
	// todo: add optional logging of requests
	// todo: optional authorization of requests

	// append the exposed thing here

}

type RCDeviceRecipe struct {
	*cells.HiveCellBase
	f         api.ICellFactory
	formation api.IRecipe
}

// Start the recipe and factory.
func (r *RCDeviceRecipe) Start() {
	r.f.Start()
}

// Stop the recipe and factory.
func (r *RCDeviceRecipe) Stop() {
	r.f.Stop()
}

// NewRCDeviceRecipe returns a recipe for a reverse-connected device.
// Intended for IoT devices that use reverse connection to a gateway or Hub.
//
// If an exposed thing is provided to the recipe it is added to the recipe chain.
// To add the device the the gateway directory, it should publish its TD on Start.
// Invoke Start on the recipe to run the device.
//
// * support HiveEnvironment commandline options
// * load CA and client certificate, and auth token if found
// * auto-discovery gateway/hub server URL if not provided
// * use gateway TD if available, fallback to serverURL scheme for protocol
// * enable auto-reconnect
// * establish client connection
//
//		f is the factory that creates the cells.
//		eThing is an optional exposed thing which will be included at the end of the
//		chain and will receive requests and emit notifications.
//	 For it to publish its TD, its sink must be set to the start of the chain.
//
// This returns the recipe, which can be used like any other cell
func NewRCDeviceRecipe(env *api.HiveEnvironment, eThing api.IHiveCell) (
	r *RCDeviceRecipe, f api.ICellFactory, err error) {

	f = factory_service.NewCellFactory(env, nil)
	chain := RCDeviceChain
	formation, err := factory_service.NewChainFormation(f, chain, eThing)
	if err != nil {
		return nil, nil, err
	}
	// requests published by the exposed thing are send back to the server
	// used for publishing TDs
	clientCell := f.GetCell(clients.TransportClientCellType)
	eThing.SetRequestSink(clientCell)

	hostname, _ := os.Hostname()
	thingID := hostname + ":device"
	r = &RCDeviceRecipe{
		HiveCellBase: cells.NewHiveCellBase(thingID),
		f:            f,
		formation:    formation,
	}

	return r, f, err
}
