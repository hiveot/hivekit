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
//
// Each of the modules can be obtained with api.GetFactoryModule[IModuleAPI](f,moduleType),
//
//	where IModuleAPI is the defined interface of the module,
//	f is the factory instance.
//	moduleType is the registration name of the module.
//
// To make the app discoverable:
// After the chain has started, the app can send an invokeaction request with the name
// 'ServeThingTDAction' and the TD/TM as the payload. The chain will update the forms with the
// server information and serve a discovery record using DNS-SD.
var StandAloneDeviceModuleChain = []api.ModuleDefinition{
	{
		// If no CA certificate is found in the AppEnvironment then generate a CA.
		// If no server certificate is found in the AppEnvironment then generate a self-signed certificate.
		Type:        certs.InitFactoryCertsModuleType,
		Constructor: certspkg.NewInitFactoryCerts,
	},

	{
		// add forms to update the published TD with appropriate forms
		Type:        addforms.AddFormsModuleType,
		Constructor: addformspkg.NewAddFormsServiceFactory,
	},
	{
		// discovery server for publishing the device TD
		// invoke action ServeThingTDAction to expose a TD
		// or locate the module and call ServeThingTD()
		// Type: discovery.DiscoveryServerModuleType,
		// Constructor: discoverypkg.NewDiscoveryServerFactory,
		Type:        discovery.ThingDiscoveryServerModuleType,
		Constructor: discoverypkg.NewThingDiscoveryServerFactory,
	},

	{
		// http server module is needed by websocket transport server
		// It uses the factory registered authenticator.
		Type:        api.HttpServerModuleType,
		Constructor: tlsserverpkg.NewTLSServerFactory,
	},
	{
		// Websocket transport server for incoming connections
		// This will be used later to update forms in the TD
		Type:        wss.HiveotWebsocketServerModuleType,
		Constructor: wsspkg.NewHiveotWssServerFactory,
	},
	{
		// Register the transport server authentication handler, and handle requests
		// to manage authentication configuration.
		Type:        authn.AuthnServiceModuleType,
		Constructor: authnpkg.NewAuthnServiceFactory,
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
// f is the module factory to use to use.
//
// This returns the recipe, which can be used like any other module
func NewStandAloneDeviceRecipe(f api.IModuleFactory) api.IRecipe {
	chain := StandAloneDeviceModuleChain

	r := factorypkg.NewChainRecipe(f, chain)
	return r
}
