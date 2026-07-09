package recipes

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/authn"
	authnpkg "github.com/hiveot/hivekit/go/modules/authn/pkg"
	"github.com/hiveot/hivekit/go/modules/authz"
	authzpkg "github.com/hiveot/hivekit/go/modules/authz/pkg"
	"github.com/hiveot/hivekit/go/modules/certs"
	certspkg "github.com/hiveot/hivekit/go/modules/certs/pkg"
	"github.com/hiveot/hivekit/go/modules/digitwin"
	digitwinpkg "github.com/hiveot/hivekit/go/modules/digitwin/pkg"
	"github.com/hiveot/hivekit/go/modules/directory"
	directorypkg "github.com/hiveot/hivekit/go/modules/directory/pkg"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/history"
	historypkg "github.com/hiveot/hivekit/go/modules/history/pkg"
	"github.com/hiveot/hivekit/go/modules/logging"
	loggingpkg "github.com/hiveot/hivekit/go/modules/logging/pkg"
	"github.com/hiveot/hivekit/go/modules/router"
	routerpkg "github.com/hiveot/hivekit/go/modules/router/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transport/discovery/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/httpbasic"
	httpbasicpkg "github.com/hiveot/hivekit/go/modules/transport/httpbasic/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/ssesc"
	ssescpkg "github.com/hiveot/hivekit/go/modules/transport/ssesc/pkg"
	tlsserverpkg "github.com/hiveot/hivekit/go/modules/transport/tlsserver/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/wss"
	wsspkg "github.com/hiveot/hivekit/go/modules/transport/wss/pkg"
)

// AppGatewayRecipe is a defines a module chain of an application gateway.
//
// # IN DEVELOPMENT - NOT READY YET
//
// The application gateway provides protocol servers, authentication, a directory,
// a router for communication with connected devices, and more.
var AppGatewayRecipeModules = []api.ModuleDefinition{
	{
		// If no CA certificate is found in the AppEnvironment then generate a CA.
		// If no server certificate is found in the AppEnvironment then generate a self-signed certificate.
		Type:        certs.InitFactoryCertsModuleType,
		Constructor: certspkg.NewInitFactoryCerts,
	},

	{
		// http server module is needed by websocket transport server
		// It uses the factory registered authenticator.
		Type:        api.HttpServerModuleType,
		Constructor: tlsserverpkg.NewTLSServerFactory,
	},
	// --- nested recipe with the servers in bus formation
	{
		// requests are passed to all servers until one accepts
		Type:        api.BusRecipeType,
		Constructor: factorypkg.NewBusRecipeFactory,
		Config: []api.ModuleDefinition{
			{
				// http-basic transport server
				Type:        httpbasic.HttpBasicServerModuleType,
				Constructor: httpbasicpkg.NewHttpBasicServerFactory,
			},
			{
				// Websocket transport server
				Type:        wss.WotWebsocketServerModuleType,
				Constructor: wsspkg.NewWotWssServerFactory,
			},
			{
				// Hiveot SSE
				Type:        ssesc.SseScServerModuleType,
				Constructor: ssescpkg.NewSseScServerFactory,
			},
			// {
			// 	// MQTT server
			// 	Type:        mqtt.MqttServerModuleType,
			// 	Constructor: mqttpkg.NewMqttServerFactory,
			// },
			// {
			// 	// MQTT gateway as client
			// 	Type:        mqttgw.MqttGatewayModuleType,
			// 	Constructor: mqttgwpkg.NewMqttGatewayFactory,
			// },
		},
	},
	{
		// logging of requests
		Type:        logging.LoggingServiceModuleType,
		Constructor: loggingpkg.NewLoggingServiceFactory,
	},
	{
		// Aerver authentication handler and service
		Type:        authn.AuthnServiceModuleType,
		Constructor: authnpkg.NewAuthnServiceFactory,
	},
	{
		// Authorization
		Type:        authz.AuthzServiceModuleType,
		Constructor: authzpkg.NewAuthzServiceFactory,
	},

	{
		// request and notification history storage
		Type:        history.HistoryModuleType,
		Constructor: historypkg.NewHistoryServiceFactory,
	},
	{
		// Directory service
		Type:        directory.DirectoryServiceModuleType,
		Constructor: directorypkg.NewDirectoryServiceFactory,
	},
	{
		// discovery of the directory
		Type:        discovery.DirectoryDiscoveryServerModuleType,
		Constructor: discoverypkg.NewDirectoryDiscoveryServerFactory,
	},

	{
		// Digitwin service slot if configured
		Type: "digitwin-slot",
	},
	{
		// Router service for routing requests to devices
		Type:        router.RouterModuleType,
		Constructor: routerpkg.NewRouterServiceFactory,
	},

	// todo: optional logging of requests
	// todo: optional authorization of requests
}

// NewAppGatewayDeviceRecipe creates a recipe for an application gateway.
//
// Intended as the central connection point for consumers and RC devices, services,
// and external devices whose TD exists in the directory.
//
// This:
// 1. manages certificates
// 2. manages users and handles authentication
// 3. provides a directory service
// 4. supports directory discovery
// 5. runs protocol servers for http-basic, websockets, grpc and others
// 6. Option to include a digital twin service
//
// Module chain:
//
//	 -> init certs
//		  -> server group [http, wss, sse, mqtt]
//		     -> logging
//		        -> authn
//		           -> authz
//		              -> history
//			              -> directory
//			                 -> discovery server
//			                    -> digitwin | vcache (optional)
//		    	                   -> router | reconnect | clients
//
// This returns the recipe, which can be used like any other module
func NewAppGatewayDeviceRecipe(f api.IModuleFactory,
	includeDigitwin bool) api.IRecipe {

	chain := AppGatewayRecipeModules
	r := factorypkg.NewChainRecipe(f, chain)

	if includeDigitwin {
		digitwinDef := api.ModuleDefinition{
			Type:        digitwin.DigitwinModuleType,
			Constructor: digitwinpkg.NewDigitwinServiceFactory,
		}
		r.SetSlot("digitwin-slot", digitwinDef)
	}
	// looks like there is work to do
	return nil
}
