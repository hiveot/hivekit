package rcdevice_recipe

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/discovery"
	discovery_client "github.com/hiveot/hivekit/go/cells/discovery/client"
	"github.com/hiveot/hivekit/go/cells/reconnect"
	reconnect_service "github.com/hiveot/hivekit/go/cells/reconnect/service"
	"github.com/hiveot/hivekit/go/cells/transport/clients"
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
		// this sets the factory directory TD
		Type:        discovery.DiscoveryClientCellType,
		Constructor: discovery_client.NewDiscoveryClientFactory,
	},
	{
		// enable auto-reconnect for the client
		Type:        reconnect.ReconnectCellType,
		Constructor: reconnect_service.NewReconnectServiceFactory,
	},
	// FIXME: if the client connection fails, the following cell will be ignored
	// and reconnect will link to the next cell, which is not a transport client.
	// as a result an error is reported that isn't the real problem.

	// {
	// 	Type:        wss.WotWebsocketClientCellType,
	// 	Constructor: wss_client.NewWotWssClientFactory,
	// },
	{
		// connect a client to the gateway/directory server.
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
	// *cells.HiveCellBase
	api.IHiveCell
	f         api.ICellFactory
	formation api.IHiveCell
	eThing    api.IHiveCell
}

// Get the client connection
func (r *RCDeviceRecipe) GetClient() api.ITransportClient {
	cell := r.f.GetCell(clients.TransportClientCellType)
	if cell == nil {
		return nil
	}
	return cell.(api.ITransportClient)
}

// Run the recipe until a signal is received
func (r *RCDeviceRecipe) Run() {
	r.Start()
	defer r.Stop()

	rcr := r.GetClient()

	fmt.Printf("Device is running. and connected with clientID '%s' to server at %v\n",
		rcr.GetClientID(), rcr.GetConnectURL())
	fmt.Println("Waiting for signal to stop...")
	r.f.WaitForSignal(context.Background())
}

// Start the recipe and factory.
func (r *RCDeviceRecipe) Start() {
	slog.Info("Starting RCDeviceRecipe", "thingID", r.GetID())
	r.f.Start()
	if r.eThing != nil {
		r.eThing.Start()
	}
}

// Stop the recipe and factory.
func (r *RCDeviceRecipe) Stop() {
	slog.Info("Stopping RCDeviceRecipe", "thingID", r.GetID())
	if r.eThing != nil {
		r.eThing.Stop()
	}
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
//	f is the factory that creates the cells.
//	eThing is an optional exposed thing which will be included at the end of the
//	 chain and will receive requests and emit notifications.
//	 It will be started when this recipe start is called.
//
// This returns the recipe, which can be used like any other cell.
func NewRCDeviceRecipe(env *api.HiveEnvironment, eThing api.IHiveCell) (
	r *RCDeviceRecipe, f api.ICellFactory, err error) {

	f = factory_service.NewCellFactory(env, nil)
	chain := RCDeviceChain
	// FIXME: recipe doesnt pass requests to chain so it can't be used
	// as the sink itself.
	formation, err := factory_service.NewChainFormation(f, chain, eThing, false)
	if err != nil {
		return nil, nil, err
	}
	// Set the transport client of the recipe as the request sink of the eThing,
	// so that it can publish its TD.
	// FIXME: this wouldnt be neccesary if this can be a notification
	if eThing != nil {
		tpClient, _ := f.GetCell(clients.TransportClientCellType).(api.ITransportClient)
		eThing.SetRequestSink(tpClient)
	}

	// FIXME: why does the device recipe need a thingID? the eThing has one...
	// hostname, _ := os.Hostname()
	// thingID := hostname + ":device-recipe"
	r = &RCDeviceRecipe{
		// HiveCellBase: formation, //cells.NewHiveCellBase(thingID),
		IHiveCell: formation,
		f:         f,
		formation: formation,
		eThing:    eThing,
	}

	return r, f, err
}
