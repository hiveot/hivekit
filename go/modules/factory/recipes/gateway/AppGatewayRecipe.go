package gatewayrecipe

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/authn"
	authn_service "github.com/hiveot/hivekit/go/modules/authn/service"
	"github.com/hiveot/hivekit/go/modules/authz"
	authz_service "github.com/hiveot/hivekit/go/modules/authz/service"
	"github.com/hiveot/hivekit/go/modules/certs"
	certs_service "github.com/hiveot/hivekit/go/modules/certs/service"
	"github.com/hiveot/hivekit/go/modules/digitwin"
	digitwin_service "github.com/hiveot/hivekit/go/modules/digitwin/service"
	"github.com/hiveot/hivekit/go/modules/directory"
	directory_service "github.com/hiveot/hivekit/go/modules/directory/service"
	factory_service "github.com/hiveot/hivekit/go/modules/factory/service"
	"github.com/hiveot/hivekit/go/modules/history"
	history_service "github.com/hiveot/hivekit/go/modules/history/service"
	"github.com/hiveot/hivekit/go/modules/logging"
	logging_service "github.com/hiveot/hivekit/go/modules/logging/service"
	"github.com/hiveot/hivekit/go/modules/router"
	router_service "github.com/hiveot/hivekit/go/modules/router/service"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	discovery_server "github.com/hiveot/hivekit/go/modules/transport/discovery/server"
	"github.com/hiveot/hivekit/go/modules/transport/httpbasic"
	httpbasic_server "github.com/hiveot/hivekit/go/modules/transport/httpbasic/server"
	"github.com/hiveot/hivekit/go/modules/transport/ssesc"
	ssesc_server "github.com/hiveot/hivekit/go/modules/transport/ssesc/server"
	tls_server "github.com/hiveot/hivekit/go/modules/transport/tlsserver/server"
	"github.com/hiveot/hivekit/go/modules/transport/wss"
	wss_server "github.com/hiveot/hivekit/go/modules/transport/wss/server"
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
		Constructor: certs_service.NewInitFactoryCerts,
	},

	{
		// http server module is needed by websocket transport server
		// It uses the factory registered authenticator.
		Type:        api.HttpServerModuleType,
		Constructor: tls_server.NewTLSServerFactory,
	},
	// --- nested recipe with the servers in bus formation
	{
		// requests are passed to all servers until one accepts
		Type:        api.BusRecipeType,
		Constructor: factory_service.NewBusRecipeFactory,
		Config: []api.ModuleDefinition{
			{
				// http-basic transport server
				Type:        httpbasic.HttpBasicServerModuleType,
				Constructor: httpbasic_server.NewHttpBasicServerFactory,
			},
			{
				// Websocket transport server
				Type:        wss.WotWebsocketServerModuleType,
				Constructor: wss_server.NewWotWssServerFactory,
			},
			{
				// Hiveot SSE
				Type:        ssesc.SseScServerModuleType,
				Constructor: ssesc_server.NewSseScServerFactory,
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
		Constructor: logging_service.NewLoggingServiceFactory,
	},
	{
		// Aerver authentication handler and service
		Type:        authn.AuthnServiceModuleType,
		Constructor: authn_service.NewAuthnServiceFactory,
	},
	{
		// Authorization
		Type:        authz.AuthzServiceModuleType,
		Constructor: authz_service.NewAuthzServiceFactory,
	},

	{
		// request and notification history storage
		Type:        history.HistoryModuleType,
		Constructor: history_service.NewHistoryServiceFactory,
	},
	{
		// Directory service
		Type:        directory.DirectoryServiceModuleType,
		Constructor: directory_service.NewDirectoryServiceFactory,
	},
	{
		// discovery of the directory
		Type:        discovery.DirectoryDiscoveryServerModuleType,
		Constructor: discovery_server.NewDirectoryDiscoveryServerFactory,
	},

	{
		// Digitwin service slot if configured
		Type: "digitwin-slot",
	},
	{
		// Router service for routing requests to devices
		Type:        router.RouterModuleType,
		Constructor: router_service.NewRouterServiceFactory,
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
	r := factory_service.NewChainRecipe(f, chain)

	if includeDigitwin {
		digitwinDef := api.ModuleDefinition{
			Type:        digitwin.DigitwinModuleType,
			Constructor: digitwin_service.NewDigitwinServiceFactory,
		}
		r.SetSlot("digitwin-slot", digitwinDef)
	}
	// looks like there is work to do
	return nil
}
