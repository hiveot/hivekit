package sadevice_recipe

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/authn"
	authn_service "github.com/hiveot/hivekit/go/cells/authn/service"
	"github.com/hiveot/hivekit/go/cells/certs"
	certs_service "github.com/hiveot/hivekit/go/cells/certs/service"
	"github.com/hiveot/hivekit/go/cells/discovery"
	discovery_server "github.com/hiveot/hivekit/go/cells/discovery/server"
	"github.com/hiveot/hivekit/go/cells/transport/addforms"
	addforms_service "github.com/hiveot/hivekit/go/cells/transport/addforms/service"
	tls_server "github.com/hiveot/hivekit/go/cells/transport/tlsserver/server"
	"github.com/hiveot/hivekit/go/cells/transport/wss"
	wss_server "github.com/hiveot/hivekit/go/cells/transport/wss/server"
	factory_service "github.com/hiveot/hivekit/go/factory/service"
)

// StandAloneDeviceChain is a template that defines the chain of cells for an IoT device
// running a server with thing discovery.
var StandAloneDeviceChain = []api.CellDefinition{
	{
		// If no CA certificate is found in the HiveEnvironment then generate a CA.
		// If no server certificate is found in the HiveEnvironment then generate a self-signed certificate.
		Type:        certs.InitFactoryCertsCellType,
		Constructor: certs_service.RunInitFactoryCerts,
	},

	// A: handle outgoing request to write device TD
	// alt: use slot for the device Thing and put this behind it.
	{
		// add forms to update the published TD with appropriate forms
		Type:        addforms.AddFormsCellType,
		Constructor: addforms_service.NewAddFormsServiceFactory,
	},
	{
		// discovery server for publishing the device TD
		// this takes the place of a directory
		Type:        discovery.DiscoveryServerCellType,
		Constructor: discovery_server.NewDiscoveryServerFactory,
	},

	// B: handle incoming request from servers
	{
		// http server is needed by websocket transport server
		// It uses the factory registered authenticator.
		Type:        api.HttpServerCellType,
		Constructor: tls_server.NewTLSServerFactory,
	},
	{
		// Websocket transport server for incoming connections
		// This will be used later to update forms in the TD
		// Tip: use BusFormation to support multiple protocols. See gateway recipe.
		Type:        wss.WotWebsocketServerCellType,
		Constructor: wss_server.NewWotWssServerFactory,
	},
	{
		// Register the transport server authentication handler.
		Type:        authn.AuthnServiceCellType,
		Constructor: authn_service.NewAuthnServiceFactory,
	},

	// consider adding logging of requests
	// consider authorization of requests

	// exposed-things are linked at the end of the chain.
}

// NewStandAloneDeviceRecipe creates a recipe for standalone IOT devices running a server.
//
// To receive requests from the recipe, provide an exposed-thing cell that handles
// the device requests and emits notifications for property updates and events.
//
// The ExposedThing cell contains the logic for publishing events, updating properties,
// and handling read requests for properties. Use of ExposedThing is optional.
//
// To publish a device TD send a createThing request to the head of the recipe,
// which forwards it to the discovery server. If an Exposed Thing is provided, its
// request sink is linked back to the chain so its PublishTD method will do this for you.
//
// Invoke Start on the recipe to run the device.
//
// 1. load CA and server certificate
// 2. Intercept updateTD and add forms to the published TD/TM
// 3. Run a service discovery server to publish the TD using the discovery specification.
//
// Service message handling
// 4. Run a http server to publish the device TD
// 5. Run the authentication server for authenticate requests and manage clients
// 6. Run a websocket server for receiving requests
//
//	f is the cell factory to use to use.
//	eThing is the optional Exposed Thing of the application. Forwarding will be disbled.
//		A call to Start and Stop will also be passed to the eThing.
//
// This returns the recipe, which can be used like any other cell.
// Call Stop to end the application.
func NewStandAloneDeviceRecipe(f api.ICellFactory, eThing api.IHiveCell) (api.IHiveCell, error) {
	chain := StandAloneDeviceChain

	r, err := factory_service.NewChainFormation(f, chain, eThing)

	// Send requests from the exposed-thing  back to the chain server, to support
	// publishing a TD to the discovery server. This disables request forwarding
	// on the eThing as unhandled requests would otherwise loop back to the chain.
	if eThing != nil {
		eThing.SetForwarding(true, false)
		eThing.SetRequestSink(r)
	}

	return r, err
}
