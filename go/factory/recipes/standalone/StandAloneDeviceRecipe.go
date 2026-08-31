package standalonerecipe

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/authn"
	authn_service "github.com/hiveot/hivekit/go/cells/authn/service"
	"github.com/hiveot/hivekit/go/cells/certs"
	certs_service "github.com/hiveot/hivekit/go/cells/certs/service"
	"github.com/hiveot/hivekit/go/cells/transport/addforms"
	addforms_service "github.com/hiveot/hivekit/go/cells/transport/addforms/service"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	discovery_server "github.com/hiveot/hivekit/go/cells/transport/discovery/server"
	tls_server "github.com/hiveot/hivekit/go/cells/transport/tlsserver/server"
	"github.com/hiveot/hivekit/go/cells/transport/wss"
	wss_server "github.com/hiveot/hivekit/go/cells/transport/wss/server"
	factory_service "github.com/hiveot/hivekit/go/factory/service"
)

// StandAloneDeviceChain is a template that defines the chain of cells for an IoT device
// running a server with thing discovery.
//
// Each of the cells can be obtained with api.GetFactoryCell[I{name}](f,cellType),
//
//	where I{name} is the defined interface of the cell,
//	f is the factory instance.
//	cellType is the registration name of the cell.
//
// To make the app discoverable:
// After the chain has started, the app can send an invokeaction request with the name
// 'ServeThingTDAction' and the TD/TM as the payload. The chain will update the forms with the
// server information and serve a discovery record using DNS-SD.
var StandAloneDeviceChain = []api.CellDefinition{
	{
		// If no CA certificate is found in the AppEnvironment then generate a CA.
		// If no server certificate is found in the AppEnvironment then generate a self-signed certificate.
		Type:        certs.InitFactoryCertsCellType,
		Constructor: certs_service.RunInitFactoryCerts,
	},

	// A: handle outgoing request to write TD
	{
		// add forms to update the published TD with appropriate forms
		Type:        addforms.AddFormsCellType,
		Constructor: addforms_service.StartAddFormsServiceFactory,
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
		// NOTE: todo: use BusFormation to support multiple protocols.
		Type:        wss.WotWebsocketServerCellType,
		Constructor: wss_server.NewWotWssServerFactory,
	},
	{
		// Register the transport server authentication handler, and handle requests
		// to manage authentication configuration.
		Type:        authn.AuthnServiceCellType,
		Constructor: authn_service.StartAuthnServiceFactory,
	},

	// todo: optional logging of requests
	// todo: optional authorization of requests

}

// NewStandAloneDeviceRecipe creates a recipe for standalone IOT devices running a server.
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
// f is the cell factory to use to use.
//
// This returns the recipe, which can be used like any other cells
func NewStandAloneDeviceRecipe(f api.ICellFactory) api.IRecipe {
	chain := StandAloneDeviceChain

	r := factory_service.NewChainFormation(f, chain)
	return r
}
