package standalonerecipe

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/authn"
	authn_service "github.com/hiveot/hivekit/go/modules/authn/service"
	"github.com/hiveot/hivekit/go/modules/certs"
	certs_service "github.com/hiveot/hivekit/go/modules/certs/service"
	factory_service "github.com/hiveot/hivekit/go/modules/factory/service"
	"github.com/hiveot/hivekit/go/modules/transport/addforms"
	addforms_service "github.com/hiveot/hivekit/go/modules/transport/addforms/service"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	discovery_server "github.com/hiveot/hivekit/go/modules/transport/discovery/server"
	tls_server "github.com/hiveot/hivekit/go/modules/transport/tlsserver/server"
	"github.com/hiveot/hivekit/go/modules/transport/wss"
	wss_server "github.com/hiveot/hivekit/go/modules/transport/wss/server"
)

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
		Constructor: certs_service.NewInitFactoryCerts,
	},

	// A: handle outgoing request to write TD
	{
		// add forms to update the published TD with appropriate forms
		Type:        addforms.AddFormsModuleType,
		Constructor: addforms_service.NewAddFormsServiceFactory,
	},
	{
		// discovery server for publishing the device TD
		Type:        discovery.ThingDiscoveryServerModuleType,
		Constructor: discovery_server.NewThingDiscoveryServerFactory,
	},

	// B: handle incoming request from servers
	{
		// http server module is needed by websocket transport server
		// It uses the factory registered authenticator.
		Type:        api.HttpServerModuleType,
		Constructor: tls_server.NewTLSServerFactory,
	},
	{
		// Websocket transport server for incoming connections
		// This will be used later to update forms in the TD
		Type:        wss.HiveotWebsocketServerModuleType,
		Constructor: wss_server.NewHiveotWssServerFactory,
	},
	{
		// Register the transport server authentication handler, and handle requests
		// to manage authentication configuration.
		Type:        authn.AuthnServiceModuleType,
		Constructor: authn_service.NewAuthnServiceFactory,
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

	r := factory_service.NewChainRecipe(f, chain)
	return r
}
