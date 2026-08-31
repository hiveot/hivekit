package factory_test

import (
	"github.com/hiveot/hivekit/go/api"
	authnapi "github.com/hiveot/hivekit/go/cells/authn"
	authn_service "github.com/hiveot/hivekit/go/cells/authn/service"
	"github.com/hiveot/hivekit/go/cells/authz"
	authz_service "github.com/hiveot/hivekit/go/cells/authz/service"
	"github.com/hiveot/hivekit/go/cells/bucketstore"
	bucketstore_service "github.com/hiveot/hivekit/go/cells/bucketstore/service"
	"github.com/hiveot/hivekit/go/cells/certs"
	certs_service "github.com/hiveot/hivekit/go/cells/certs/service"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/cells/digitwin"
	digitwin_service "github.com/hiveot/hivekit/go/cells/digitwin/service"
	"github.com/hiveot/hivekit/go/cells/directory"
	directory_client "github.com/hiveot/hivekit/go/cells/directory/client"
	directory_service "github.com/hiveot/hivekit/go/cells/directory/service"
	"github.com/hiveot/hivekit/go/cells/history"
	history_client "github.com/hiveot/hivekit/go/cells/history/client"
	history_service "github.com/hiveot/hivekit/go/cells/history/service"
	"github.com/hiveot/hivekit/go/cells/logging"
	logging_service "github.com/hiveot/hivekit/go/cells/logging/service"
	"github.com/hiveot/hivekit/go/cells/reconnect"
	reconnect_service "github.com/hiveot/hivekit/go/cells/reconnect/service"
	"github.com/hiveot/hivekit/go/cells/router"
	router_service "github.com/hiveot/hivekit/go/cells/router/service"
	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/hiveot/hivekit/go/cells/transport/addforms"
	addforms_service "github.com/hiveot/hivekit/go/cells/transport/addforms/service"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	discovery_client "github.com/hiveot/hivekit/go/cells/transport/discovery/client"
	discovery_server "github.com/hiveot/hivekit/go/cells/transport/discovery/server"
	grpctransport "github.com/hiveot/hivekit/go/cells/transport/grpc"
	grpc_client "github.com/hiveot/hivekit/go/cells/transport/grpc/client"
	grpc_server "github.com/hiveot/hivekit/go/cells/transport/grpc/server"
	"github.com/hiveot/hivekit/go/cells/transport/httpbasic"
	httpbasic_server "github.com/hiveot/hivekit/go/cells/transport/httpbasic/server"
	"github.com/hiveot/hivekit/go/cells/transport/ssesc"
	ssesc_client "github.com/hiveot/hivekit/go/cells/transport/ssesc/client"
	ssesc_server "github.com/hiveot/hivekit/go/cells/transport/ssesc/server"
	tls_server "github.com/hiveot/hivekit/go/cells/transport/tlsserver/server"
	wss "github.com/hiveot/hivekit/go/cells/transport/wss"
	wss_client "github.com/hiveot/hivekit/go/cells/transport/wss/client"
	wss_server "github.com/hiveot/hivekit/go/cells/transport/wss/server"
	"github.com/hiveot/hivekit/go/cells/vcache"
	vcache_service "github.com/hiveot/hivekit/go/cells/vcache/service"
)

// List hivekit available cells
var HiveKitCells = []api.CellDefinition{

	//--- factory related cells

	// recipe cells - for future consideration is to embed a recipe in a recipe
	// {
	// 	Type:        factory.ChainRecipeCellType,
	// 	Constructor: factorypkg.NewChainRecipeFactory,
	// },
	// {
	// 	Type:        factory.StarRecipeCellType,
	// 	Constructor: factorypkg.NewStarRecipeFactory,
	// },

	//--- transport clients and servers ---

	// discovery transport
	{
		Type:        discovery.DiscoveryClientCellType,
		Constructor: discovery_client.StartDiscoveryClientFactory,
	},
	{
		Type:        discovery.DiscoveryServerCellType,
		Constructor: discovery_server.NewDiscoveryServerFactory,
	},
	// gRPC transport
	{
		Type:        grpctransport.HiveotGrpcClientCellType,
		Constructor: grpc_client.StartHiveotGrpcClientFactory,
	},
	{
		Type:        grpctransport.HiveotGrpcServerCellType,
		Constructor: grpc_server.StartHiveotGrpcServerFactory,
	},
	// http server provider
	{
		Type:        api.HttpServerCellType,
		Constructor: tls_server.NewTLSServerFactory,
	},
	// http-basic transport
	// {
	// 	Type:        httpbasic.HttpBasicClientCellType,
	// 	Constructor: httpbasic_client.NewHttpBasicClientFactory,
	// },
	{
		Type:        httpbasic.HttpBasicServerCellType,
		Constructor: httpbasic_server.StartHttpBasicServerFactory,
	},
	// sse-sc transport
	{
		Type:        ssesc.SseScServerCellType,
		Constructor: ssesc_server.StartSseScServerFactory,
	},
	{
		Type:        ssesc.SseScClientCellType,
		Constructor: ssesc_client.StartSseScClientFactory,
	},
	// wss transport for hiveot RRN messaging
	{
		Type:        wss.HiveotWebsocketClientCellType,
		Constructor: wss_client.StartHiveotWssClientFactory,
	},
	{
		Type:        wss.HiveotWebsocketServerCellType,
		Constructor: wss_server.NewHiveotWssServerFactory,
	},
	// wss transport for WoT websocket messaging
	{
		Type:        wss.WotWebsocketClientCellType,
		Constructor: wss_client.StartWotWssClientFactory,
	},
	{
		Type:        wss.WotWebsocketServerCellType,
		Constructor: wss_server.NewWotWssServerFactory,
	},

	//--- services ---

	// add forms to createTD or updateTD requests
	{
		Type:        addforms.AddFormsCellType,
		Constructor: addforms_service.StartAddFormsServiceFactory,
	},

	// thing service helper
	{
		Type:        thing.ExposedThingCellType,
		Constructor: thing.StartExposedThingFactory,
	},

	// client and session management provider
	{
		Type:        authnapi.AuthnServiceCellType,
		Constructor: authn_service.StartAuthnServiceFactory,
	},
	// authorization provider
	{
		Type:        authz.AuthzServiceCellType,
		Constructor: authz_service.StartAuthzServiceFactory,
	},
	// bucket store as a service
	{
		Type:        bucketstore.BucketStoreCellType,
		Constructor: bucketstore_service.StartBucketStoreServiceFactory,
	},
	// certs service
	{
		Type:        certs.CertsServiceCellType,
		Constructor: certs_service.StartCertsServiceFactory,
	},
	// InitFactoryCerts ensure the factory has certificates needed to run.
	{
		Type:        certs.InitFactoryCertsCellType,
		Constructor: certs_service.RunInitFactoryCerts,
	},
	// consumer helper
	{
		Type:        consumer.ConsumerCellType,
		Constructor: consumer.NewConsumerFactory,
	},

	// digitwin service
	{
		Type:        digitwin.DigitwinCellType,
		Constructor: digitwin_service.StartDigitwinServiceFactory,
	},
	// directory service
	{
		Type:        directory.DirectoryServiceCellType,
		Constructor: directory_service.StartDirectoryServiceFactory,
	},
	{
		Type:        directory.DirectoryClientCellType,
		Constructor: directory_client.StartDirectoryClientFactory,
	},
	// history service provider
	{
		Type:        history.HistoryServiceCellType,
		Constructor: history_service.StartHistoryServiceFactory,
	},
	{
		Type:        history.ReadHistoryClientCellType,
		Constructor: history_client.NewReadHistoryClientFactory,
	},
	// logging service provider
	{
		Type:        logging.LoggingServiceCellType,
		Constructor: logging_service.StartLoggingServiceFactory,
	},
	// auto-reconnect client
	{
		Type:        reconnect.ReconnectCellType,
		Constructor: reconnect_service.StartReconnectFactory,
	},
	// router service provider
	{
		Type:        router.RouterCellType,
		Constructor: router_service.StartRouterServiceFactory,
	},
	// vcache server provider
	{
		Type:        vcache.ValueCacheCellType,
		Constructor: vcache_service.StartValueCacheServiceFactory,
	},
}
