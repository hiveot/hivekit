package recipes

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/authn"
	authnpkg "github.com/hiveot/hivekit/go/modules/authn/pkg"
	"github.com/hiveot/hivekit/go/modules/certs"
	certspkg "github.com/hiveot/hivekit/go/modules/certs/pkg"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/addforms"
	addformspkg "github.com/hiveot/hivekit/go/modules/transport/addforms/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transport/discovery/pkg"
	tlsserverpkg "github.com/hiveot/hivekit/go/modules/transport/tlsserver/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/wss"
	wsspkg "github.com/hiveot/hivekit/go/modules/transport/wss/pkg"
)

// module type name of the slot where to insert the 'exposed thing' application module.
const AppSlotType = "appSlot"

// StandAloneDeviceModuleChain is a template that defines the module chain for an IoT device
// running a server.
var StandAloneDeviceModuleChain = []api.ModuleDefinition{

	//--- modules that do not depend on where they are placed
	{
		// initialize CA and server certs
		Type:        certs.InitFactoryCertsModuleType,
		Constructor: certspkg.NewInitFactoryCerts,
	},
	{
		// http server module is used by websockets
		Type:        api.HttpServerModuleType,
		Constructor: tlsserverpkg.NewTLSServerFactory,
	},

	// FIXME: requests to the chain are passed to the wss server which
	// tries to send it to a RC device instead of down the chain... oops

	//--- sequence required for processing requests
	{
		// websocket server transport for consumer connections
		Type:        wss.HiveotWebsocketServerModuleType,
		Constructor: wsspkg.NewHiveotWssServerFactory,
	},
	{
		// run an authentication service
		Type:        authn.AuthnServiceModuleType,
		Constructor: authnpkg.NewAuthnServiceFactory,
	},

	// todo: add optional logging of requests
	// todo: optional authorization of requests

	{
		// Module slot for the application module.
		// This is the application module. This place lets it publish its TD for discovery as it is
		// placed before those modules.
		// Use Chain.SetSlot(AppSlotType, moduleDef)
		Type: AppSlotType,
	},

	{
		// add forms to update the published TD with appropriate forms
		Type:        addforms.AddFormsModuleType,
		Constructor: addformspkg.NewAddFormsServiceFactory,
	},
	{
		// discovery server for publishing the device TD
		Type:        discovery.DiscoveryServerModuleType,
		Constructor: discoverypkg.NewDiscoveryServerFactory,
	},
}

// NewStandAloneDeviceRecipe creates a recipe for standalone IOT devices running a server.
//
// 1. load CA and server certificate
// 2. Run a http server to publish the device TD
// 3. Run the authentication server for authenticate requests and manage clients
// 4. Run a websocket server for receiving requests
// 5. {the application slot>
// 6. Add forms to the published TD/TM
// 7. Run a service discovery server to publish the TD using the discovery specification.
//
// f is the module factory to use to use.
// appModule is the module definition of the exposed thing to inject in the app slot.
//
// This returns the recipe, which can be used like any other module
func NewStandAloneDeviceRecipe(
	f api.IModuleFactory, appModule *api.ModuleDefinition) api.IRecipe {
	chain := StandAloneDeviceModuleChain

	r := factorypkg.NewChainRecipe(f, chain)
	// place the application module before discovery
	if appModule != nil {
		r.SetSlot(AppSlotType, *appModule)
	}
	return r
}
