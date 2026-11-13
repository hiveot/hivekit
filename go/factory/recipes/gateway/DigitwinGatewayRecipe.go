package gateway_recipe

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
	"github.com/hiveot/hivekit/go/cells/discovery"
	discovery_server "github.com/hiveot/hivekit/go/cells/discovery/server"
	"github.com/hiveot/hivekit/go/cells/history"
	history_service "github.com/hiveot/hivekit/go/cells/history/service"
	"github.com/hiveot/hivekit/go/cells/logging"
	logging_service "github.com/hiveot/hivekit/go/cells/logging/service"
	"github.com/hiveot/hivekit/go/cells/rcrouter"
	rcrouter_service "github.com/hiveot/hivekit/go/cells/rcrouter/service"
	"github.com/hiveot/hivekit/go/cells/router"
	router_service "github.com/hiveot/hivekit/go/cells/router/service"
	grpc "github.com/hiveot/hivekit/go/cells/transport/grpc"
	grpc_server "github.com/hiveot/hivekit/go/cells/transport/grpc/server"
	"github.com/hiveot/hivekit/go/cells/transport/httpbasic"
	httpbasic_server "github.com/hiveot/hivekit/go/cells/transport/httpbasic/server"
	"github.com/hiveot/hivekit/go/cells/transport/tlsserver"
	"github.com/hiveot/hivekit/go/cells/transport/wss"
	wss_server "github.com/hiveot/hivekit/go/cells/transport/wss/server"
	factory_service "github.com/hiveot/hivekit/go/factory/service"
)

// DigitwinGatewayRecipeCells defines a recipe for running a digital twin
// gateway.
//
// # IN DEVELOPMENT - NOT READY YET
//
// The digitwin gateway provides digital twin of devices include supporting
// services such as transport protocol servers, authentication, a directory,
// and a router for communication with connected devices.
var DigitwinGatewayRecipeCells = []api.CellDefinition{
	{
		// If no CA certificate is found in the AppEnvironment then generate a CA.
		// If no server certificate is found in the AppEnvironment then generate a self-signed certificate.
		Type:        certs.CertsServiceCellType,
		Constructor: certs_service.NewCertsServiceFactory,
		// just initialization, linking of this instance happens elsewhere
		NoLink: true,
	},
	{
		// http server is needed by websocket transport server
		// It uses the factory registered authenticator.
		Type:        api.HttpServerCellType,
		Constructor: tlsserver.NewTLSServerFactory,
	},
	// --- nested formation with the servers operating in parallel
	{
		// Note: notifications received from the server(s) - primarily RC connections-
		// should go to the router, which also receives notifications from SA devices
		// throught its client connections.
		//
		// The recipe therefore sets the bus formation notification sink to the router,
		// instead of passing it up the chain to the http server, which would discard them.

		// requests are passed to all servers until one accepts
		Type:        api.BusFormationType,
		Constructor: factory_service.NewBusFormationFactory,
		Config: []api.CellDefinition{
			{
				// Websocket transport server is the preferred transport
				Type:        wss.WotWebsocketServerCellType,
				Constructor: wss_server.NewWotWssServerFactory,
			},
			// {
			// 	// Hiveot SSE is a theoretical option, but why bother
			// 	Type:        ssesc.SseScServerCellType,
			// 	Constructor: ssesc_server.StartSseScServerFactory,
			// },
			{
				// Hiveot gRPC for fast inter-process connections
				Type:        grpc.HiveotGrpcServerCellType,
				Constructor: grpc_server.NewHiveotGrpcServerFactory,
			},
			// {
			// 	// MQTT server is coming soon
			// 	Type:        mqtt.MqttServerCellType,
			// 	Constructor: mqttpkg.NewMqttServerFactory,
			// },
			{
				// Last, the gateway supports http-basic
				Type:        httpbasic.HttpBasicServerCellType,
				Constructor: httpbasic_server.NewHttpBasicServerFactory,
			},
		},
	},
	{ // logging of requests
		Type:        logging.LoggingServiceCellType,
		Constructor: logging_service.NewLoggingServiceFactory,
	},
	{ // Authorization of remote requests
		Type:        authz.AuthzServiceCellType,
		Constructor: authz_service.NewAuthzServiceFactory,
	},
	{ // Store request and notification history
		Type:        history.HistoryServiceCellType,
		Constructor: history_service.NewHistoryServiceFactory,
	},
	{ // Authentication user handler and service
		Type:        authn.AuthnServiceCellType,
		Constructor: authn_service.NewAuthnServiceFactory,
	},

	// --- star formation with local services
	// requests are directed to the service matching the cell with the request thingID
	// This doesn't work for cells that provide multiple services.
	{
		Type:        api.StarFormationType,
		Constructor: factory_service.NewStarFormationFactory,
		Config: []api.CellDefinition{
			{ // Certificate management
				Type:        certs.CertsServiceCellType,
				Constructor: certs_service.NewCertsServiceFactory,
			},
			// other embedded services can be added here
		},
	},

	{
		// Directory service
		Type:        directory.DirectoryServiceCellType,
		Constructor: directory_service.NewDirectoryServiceFactory,
	},
	{
		// discovery of the directory (must be placed after directory)
		Type:        discovery.DiscoveryServerCellType,
		Constructor: discovery_server.NewDiscoveryServerFactory,
	},
	// The digital twin service handles requests for digital device twins
	{
		Type:        digitwin.DigitwinCellType,
		Constructor: digitwin_service.NewDigitwinServiceFactory,
	},

	{
		// RC router service for routing requests to reverse connections.
		// This routes the request to the client that send the TD update.
		Type:        rcrouter.RCRouterCellType,
		Constructor: rcrouter_service.NewRCRouterServiceFactory,
	},
	{
		// Router service for routing requests to stand-alone devices.
		// this uses the directory to lookup thing TDs.
		Type:        router.RouterCellType,
		Constructor: router_service.NewRouterServiceFactory,
	},
	// todo: optional logging of requests
	// todo: optional authorization of requests
}

// StartDigitwinGatewayRecipe creates a recipe for an IoT gateway that
// serves digital twins of devices.
//
// Invoke Start on the recipe to run the application.
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
//	-> init certs
//		-> server group [http, wss, sse, mqtt]
//			-> logging
//		    	-> authz
//		        	-> history
//						-> authn
//			              -> directory
//			                 -> discovery server
//			                    -> digitwin | vcache
//		    	                   -> router | reconnect | clients
//
// This returns the recipe, which can be used like any other cell
func StartDigitwinGatewayRecipe(f api.ICellFactory) (api.IHiveCell, error) {

	chain := DigitwinGatewayRecipeCells
	r, err := factory_service.NewChainFormation(f, chain, nil, false)

	return r, err
}
