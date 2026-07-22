package factory_test

import (
	"github.com/hiveot/hivekit/go/api"
	authnapi "github.com/hiveot/hivekit/go/modules/authn"
	authn_service "github.com/hiveot/hivekit/go/modules/authn/service"
	"github.com/hiveot/hivekit/go/modules/authz"
	authz_service "github.com/hiveot/hivekit/go/modules/authz/service"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketstore_service "github.com/hiveot/hivekit/go/modules/bucketstore/service"
	"github.com/hiveot/hivekit/go/modules/certs"
	certs_service "github.com/hiveot/hivekit/go/modules/certs/service"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/modules/digitwin"
	digitwin_service "github.com/hiveot/hivekit/go/modules/digitwin/service"
	"github.com/hiveot/hivekit/go/modules/directory"
	directory_client "github.com/hiveot/hivekit/go/modules/directory/client"
	directory_service "github.com/hiveot/hivekit/go/modules/directory/service"
	"github.com/hiveot/hivekit/go/modules/history"
	history_client "github.com/hiveot/hivekit/go/modules/history/client"
	history_service "github.com/hiveot/hivekit/go/modules/history/service"
	"github.com/hiveot/hivekit/go/modules/logging"
	logging_service "github.com/hiveot/hivekit/go/modules/logging/service"
	"github.com/hiveot/hivekit/go/modules/reconnect"
	reconnect_service "github.com/hiveot/hivekit/go/modules/reconnect/service"
	"github.com/hiveot/hivekit/go/modules/router"
	router_service "github.com/hiveot/hivekit/go/modules/router/service"
	"github.com/hiveot/hivekit/go/modules/thing"
	"github.com/hiveot/hivekit/go/modules/transport/addforms"
	addforms_service "github.com/hiveot/hivekit/go/modules/transport/addforms/service"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	discovery_client "github.com/hiveot/hivekit/go/modules/transport/discovery/client"
	discovery_server "github.com/hiveot/hivekit/go/modules/transport/discovery/server"
	grpctransport "github.com/hiveot/hivekit/go/modules/transport/grpc"
	grpc_client "github.com/hiveot/hivekit/go/modules/transport/grpc/client"
	grpc_server "github.com/hiveot/hivekit/go/modules/transport/grpc/server"
	"github.com/hiveot/hivekit/go/modules/transport/httpbasic"
	httpbasic_client "github.com/hiveot/hivekit/go/modules/transport/httpbasic/client"
	httpbasic_server "github.com/hiveot/hivekit/go/modules/transport/httpbasic/server"
	"github.com/hiveot/hivekit/go/modules/transport/ssesc"
	ssesc_client "github.com/hiveot/hivekit/go/modules/transport/ssesc/client"
	ssesc_server "github.com/hiveot/hivekit/go/modules/transport/ssesc/server"
	tls_server "github.com/hiveot/hivekit/go/modules/transport/tlsserver/server"
	wss "github.com/hiveot/hivekit/go/modules/transport/wss"
	wss_client "github.com/hiveot/hivekit/go/modules/transport/wss/client"
	wss_server "github.com/hiveot/hivekit/go/modules/transport/wss/server"
	"github.com/hiveot/hivekit/go/modules/vcache"
	vcache_service "github.com/hiveot/hivekit/go/modules/vcache/service"
)

// List hivekit available modules
var HiveKitModules = []api.ModuleDefinition{

	//--- factory related modules

	// recipe modules - for future consideration is to embed a recipe in a recipe
	// {
	// 	Type:        factory.ChainRecipeModuleType,
	// 	Constructor: factorypkg.NewChainRecipeFactory,
	// },
	// {
	// 	Type:        factory.StarRecipeModuleType,
	// 	Constructor: factorypkg.NewStarRecipeFactory,
	// },

	//--- transport module ---

	// discovery transport
	{
		Type:        discovery.DiscoveryClientModuleType,
		Constructor: discovery_client.NewDiscoveryClientFactory,
	},
	{
		Type:        discovery.DirectoryDiscoveryServerModuleType,
		Constructor: discovery_server.NewDirectoryDiscoveryServerFactory,
	},
	{
		Type:        discovery.ThingDiscoveryServerModuleType,
		Constructor: discovery_server.NewThingDiscoveryServerFactory,
	},
	// gRPC transport
	{
		Type:        grpctransport.HiveotGrpcClientModuleType,
		Constructor: grpc_client.NewHiveotGrpcClientFactory,
	},
	{
		Type:        grpctransport.HiveotGrpcServerModuleType,
		Constructor: grpc_server.NewHiveotGrpcServerFactory,
	},
	// http server provider
	{
		Type:        api.HttpServerModuleType,
		Constructor: tls_server.NewTLSServerFactory,
	},
	// http-basic transport
	{
		Type:        httpbasic.HttpBasicClientModuleType,
		Constructor: httpbasic_client.NewHttpBasicClientFactory,
	},
	{
		Type:        httpbasic.HttpBasicServerModuleType,
		Constructor: httpbasic_server.NewHttpBasicServerFactory,
	},
	// sse-sc transport
	{
		Type:        ssesc.SseScServerModuleType,
		Constructor: ssesc_server.NewSseScServerFactory,
	},
	{
		Type:        ssesc.SseScClientModuleType,
		Constructor: ssesc_client.NewSseScClientFactory,
	},
	// wss transport for hiveot RRN messaging
	{
		Type:        wss.HiveotWebsocketClientModuleType,
		Constructor: wss_client.NewHiveotWssClientFactory,
	},
	{
		Type:        wss.HiveotWebsocketServerModuleType,
		Constructor: wss_server.NewHiveotWssServerFactory,
	},
	// wss transport for WoT websocket messaging
	{
		Type:        wss.WotWebsocketClientModuleType,
		Constructor: wss_client.NewWotWssClientFactory,
	},
	{
		Type:        wss.WotWebsocketServerModuleType,
		Constructor: wss_server.NewWotWssServerFactory,
	},

	//--- services ---

	// add forms to createTD or updateTD requests
	{
		Type:        addforms.AddFormsModuleType,
		Constructor: addforms_service.NewAddFormsServiceFactory,
	},

	// thing service helper
	{
		Type:        thing.ExposedThingModuleType,
		Constructor: thing.NewExposedThingFactory,
	},

	// client and session management provider
	{
		Type:        authnapi.AuthnServiceModuleType,
		Constructor: authn_service.NewAuthnServiceFactory,
	},
	// authorization provider
	{
		Type:        authz.AuthzServiceModuleType,
		Constructor: authz_service.NewAuthzServiceFactory,
	},
	// bucket store as a service
	{
		Type:        bucketstore.BucketStoreModuleType,
		Constructor: bucketstore_service.NewBucketStoreServiceFactory,
	},
	// certs service
	{
		Type:        certs.CertsServerModuleType,
		Constructor: certs_service.NewCertsServiceFactory,
	},
	// InitFactoryCerts ensure the factory has certificates needed to run.
	{
		Type:        certs.InitFactoryCertsModuleType,
		Constructor: certs_service.NewInitFactoryCerts,
	},
	// consumer helper
	{
		Type:        consumer.ConsumerModuleType,
		Constructor: consumer.NewConsumerFactory,
	},

	// digitwin service
	{
		Type:        digitwin.DigitwinModuleType,
		Constructor: digitwin_service.NewDigitwinServiceFactory,
	},
	// directory service
	{
		Type:        directory.DirectoryServiceModuleType,
		Constructor: directory_service.NewDirectoryServiceFactory,
	},
	{
		Type:        directory.DirectoryClientModuleType,
		Constructor: directory_client.NewDirectoryClientFactory,
	},
	// history service provider
	{
		Type:        history.HistoryModuleType,
		Constructor: history_service.NewHistoryServiceFactory,
	},
	{
		Type:        history.ReadHistoryClientModuleType,
		Constructor: history_client.NewReadHistoryClientFactory,
	},
	// logging service provider
	{
		Type:        logging.LoggingServiceModuleType,
		Constructor: logging_service.NewLoggingServiceFactory,
	},
	// auto-reconnect client
	{
		Type:        reconnect.ReconnectModuleType,
		Constructor: reconnect_service.NewReconnectFactory,
	},
	// router service provider
	{
		Type:        router.RouterModuleType,
		Constructor: router_service.NewRouterServiceFactory,
	},
	// vcache server provider
	{
		Type:        vcache.ValueCacheModuleType,
		Constructor: vcache_service.NewValueCacheServiceFactory,
	},
}
