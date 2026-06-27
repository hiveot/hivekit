package recipes

import (
	"github.com/hiveot/hivekit/go/modules/authn"
	authnpkg "github.com/hiveot/hivekit/go/modules/authn/pkg"
	"github.com/hiveot/hivekit/go/modules/certs"
	certspkg "github.com/hiveot/hivekit/go/modules/certs/pkg"
	"github.com/hiveot/hivekit/go/modules/factory"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/addforms"
	addformspkg "github.com/hiveot/hivekit/go/modules/transport/addforms/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transport/discovery/pkg"
	tlsserverpkg "github.com/hiveot/hivekit/go/modules/transport/tlsserver/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/wss"
	wsspkg "github.com/hiveot/hivekit/go/modules/transport/wss/pkg"
)

// the module slot where to insert the application exposed thing
const AppSlotType = "appSlot"

// ServerDeviceModuleChain is a template that defines the module chain for an IoT device
// running a server.
var ServerDeviceModuleChain = []factory.ModuleDefinition{

	//--- modules that do not depend on where they are placed
	{
		// initialize CA and server certs
		Type:        certs.InitFactoryCertsModuleType,
		Constructor: certspkg.NewInitFactoryCerts,
	},
	{
		// http server module is used by websockets
		Type:        transport.TLSServerModuleType,
		Constructor: tlsserverpkg.NewTLSServerFactory,
	},
	{
		// run an authentication service
		Type:        authn.AuthnServiceModuleType,
		Constructor: authnpkg.NewAuthnServiceFactory,
	},

	//--- sequence required for processing requests
	{
		// websockets is the main communication transport
		Type:        wss.HiveotWebsocketServerModuleType,
		Constructor: wsspkg.NewHiveotWssServerFactory,
	},

	// todo: add optional logging of requests
	// todo: optional authorization of requests

	{
		// Module slot for the application module.
		// Use this slot to allow modules to publish their TD for discovery as it is
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

// NewServerDeviceRecipe creates a recipe for IOT devices running a server.
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
func NewServerDeviceRecipe(
	f factory.IModuleFactory, appModule *factory.ModuleDefinition) factory.IRecipe {
	chain := ServerDeviceModuleChain

	r := factorypkg.NewChainRecipe(f, chain)
	// place the application module before discovery
	if appModule != nil {
		r.SetSlot(AppSlotType, *appModule)
	}
	return r
}
