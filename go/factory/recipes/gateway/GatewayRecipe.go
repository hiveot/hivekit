package gatewayrecipe

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/authn"
	authn_service "github.com/hiveot/hivekit/go/cells/authn/service"
	"github.com/hiveot/hivekit/go/cells/authz"
	authz_service "github.com/hiveot/hivekit/go/cells/authz/service"
	"github.com/hiveot/hivekit/go/cells/certs"
	certs_service "github.com/hiveot/hivekit/go/cells/certs/service"
	"github.com/hiveot/hivekit/go/cells/digitwin"
	digitwin_service "github.com/hiveot/hivekit/go/cells/digitwin/service"
	"github.com/hiveot/hivekit/go/cells/directory"
	directory_service "github.com/hiveot/hivekit/go/cells/directory/service"
	"github.com/hiveot/hivekit/go/cells/history"
	history_service "github.com/hiveot/hivekit/go/cells/history/service"
	"github.com/hiveot/hivekit/go/cells/logging"
	logging_service "github.com/hiveot/hivekit/go/cells/logging/service"
	"github.com/hiveot/hivekit/go/cells/router"
	router_service "github.com/hiveot/hivekit/go/cells/router/service"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	discovery_server "github.com/hiveot/hivekit/go/cells/transport/discovery/server"
	grpc "github.com/hiveot/hivekit/go/cells/transport/grpc"
	grpc_server "github.com/hiveot/hivekit/go/cells/transport/grpc/server"
	"github.com/hiveot/hivekit/go/cells/transport/httpbasic"
	httpbasic_server "github.com/hiveot/hivekit/go/cells/transport/httpbasic/server"
	"github.com/hiveot/hivekit/go/cells/transport/ssesc"
	ssesc_server "github.com/hiveot/hivekit/go/cells/transport/ssesc/server"
	tls_server "github.com/hiveot/hivekit/go/cells/transport/tlsserver/server"
	"github.com/hiveot/hivekit/go/cells/transport/wss"
	wss_server "github.com/hiveot/hivekit/go/cells/transport/wss/server"
	factory_service "github.com/hiveot/hivekit/go/factory/service"
)

// AppGatewayRecipe is a defines a cell chain of an application gateway.
//
// # IN DEVELOPMENT - NOT READY YET
//
// The application gateway provides protocol servers, authentication, a directory,
// a router for communication with connected devices, and more.
var AppGatewayRecipeCells = []api.CellDefinition{
	{
		// If no CA certificate is found in the AppEnvironment then generate a CA.
		// If no server certificate is found in the AppEnvironment then generate a self-signed certificate.
		Type:        certs.InitFactoryCertsCellType,
		Constructor: certs_service.RunInitFactoryCerts,
	},
	{
		// http server is needed by websocket transport server
		// It uses the factory registered authenticator.
		Type:        api.HttpServerCellType,
		Constructor: tls_server.NewTLSServerFactory,
	},
	// --- nested recipe with the servers operating in parallel
	{
		// requests are passed to all servers until one accepts
		Type:        api.BusRecipeType,
		Constructor: factory_service.NewBusFactory,
		Config: []api.CellDefinition{
			{
				// http-basic transport server
				Type:        httpbasic.HttpBasicServerCellType,
				Constructor: httpbasic_server.StartHttpBasicServerFactory,
			},
			{
				// Websocket transport server
				Type:        wss.WotWebsocketServerCellType,
				Constructor: wss_server.NewWotWssServerFactory,
			},
			{
				// Hiveot SSE
				Type:        ssesc.SseScServerCellType,
				Constructor: ssesc_server.StartSseScServerFactory,
			},
			{
				// Hiveot gRPC
				Type:        grpc.HiveotGrpcServerCellType,
				Constructor: grpc_server.StartHiveotGrpcServerFactory,
			},
			// {
			// 	// MQTT server
			// 	Type:        mqtt.MqttServerCellType,
			// 	Constructor: mqttpkg.NewMqttServerFactory,
			// },
			// {
			// 	// MQTT client
			// 	Type:        mqttgw.MqttClientCellType,
			// 	Constructor: mqttgwpkg.NewMqttClientFactory,
			// },
		},
	},
	{
		// logging of requests
		Type:        logging.LoggingServiceCellType,
		Constructor: logging_service.StartLoggingServiceFactory,
	},
	{
		// Authentication handler and service
		Type:        authn.AuthnServiceCellType,
		Constructor: authn_service.StartAuthnServiceFactory,
	},
	{
		// Authorization
		Type:        authz.AuthzServiceCellType,
		Constructor: authz_service.StartAuthzServiceFactory,
	},

	{
		// request and notification history storage
		Type:        history.HistoryServiceCellType,
		Constructor: history_service.StartHistoryServiceFactory,
	},
	{
		// Directory service
		Type:        directory.DirectoryServiceCellType,
		Constructor: directory_service.StartDirectoryServiceFactory,
	},
	{
		// discovery of the directory (must be placed after directory)
		Type:        discovery.DiscoveryServerCellType,
		Constructor: discovery_server.NewDiscoveryServerFactory,
	},

	{
		// Digitwin service slot if configured
		Type: "digitwin-slot",
	},
	{
		// Router service for routing requests to devices
		// this requires a directory client or service.
		Type:        router.RouterCellType,
		Constructor: router_service.StartRouterServiceFactory,
	},

	// todo: optional logging of requests
	// todo: optional authorization of requests
}

// NewGatewayDeviceRecipe creates a recipe for an IoT gateway.
//
// Intended as the central connection point for consumers, services, RC devices,
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
// Cell chain:
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
// This returns the recipe, which can be used like any other cell
func NewGatewayDeviceRecipe(f api.ICellFactory, includeDigitwin bool) api.IRecipe {

	chain := AppGatewayRecipeCells
	r := factory_service.NewChainFormation(f, chain)

	if includeDigitwin {
		digitwinDef := api.CellDefinition{
			Type:        digitwin.DigitwinCellType,
			Constructor: digitwin_service.StartDigitwinServiceFactory,
		}
		r.SetSlot("digitwin-slot", digitwinDef)
	}
	return r
}
