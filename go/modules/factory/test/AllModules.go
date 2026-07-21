package factory_test

import (
	"github.com/hiveot/hivekit/go/api"
	authnapi "github.com/hiveot/hivekit/go/modules/authn"
	authnpkg "github.com/hiveot/hivekit/go/modules/authn/pkg"
	"github.com/hiveot/hivekit/go/modules/authz"
	authzpkg "github.com/hiveot/hivekit/go/modules/authz/pkg"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketstorepkg "github.com/hiveot/hivekit/go/modules/bucketstore/pkg"
	"github.com/hiveot/hivekit/go/modules/certs"
	certspkg "github.com/hiveot/hivekit/go/modules/certs/pkg"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/modules/digitwin"
	digitwinpkg "github.com/hiveot/hivekit/go/modules/digitwin/pkg"
	"github.com/hiveot/hivekit/go/modules/directory"
	directorypkg "github.com/hiveot/hivekit/go/modules/directory/pkg"
	"github.com/hiveot/hivekit/go/modules/history"
	historypkg "github.com/hiveot/hivekit/go/modules/history/pkg"
	"github.com/hiveot/hivekit/go/modules/logging"
	loggingpkg "github.com/hiveot/hivekit/go/modules/logging/pkg"
	"github.com/hiveot/hivekit/go/modules/reconnect"
	reconnectpkg "github.com/hiveot/hivekit/go/modules/reconnect/pkg"
	"github.com/hiveot/hivekit/go/modules/router"
	routerpkg "github.com/hiveot/hivekit/go/modules/router/pkg"
	"github.com/hiveot/hivekit/go/modules/thing"
	"github.com/hiveot/hivekit/go/modules/transport/addforms"
	addformspkg "github.com/hiveot/hivekit/go/modules/transport/addforms/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transport/discovery/pkg"
	grpctransport "github.com/hiveot/hivekit/go/modules/transport/grpc"
	grpcpkg "github.com/hiveot/hivekit/go/modules/transport/grpc/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/httpbasic"
	httpbasicpkg "github.com/hiveot/hivekit/go/modules/transport/httpbasic/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/ssesc"
	ssescpkg "github.com/hiveot/hivekit/go/modules/transport/ssesc/pkg"
	tlsserverpkg "github.com/hiveot/hivekit/go/modules/transport/tlsserver/pkg"
	wss "github.com/hiveot/hivekit/go/modules/transport/wss"
	wsspkg "github.com/hiveot/hivekit/go/modules/transport/wss/pkg"
	"github.com/hiveot/hivekit/go/modules/vcache"
	vcachepkg "github.com/hiveot/hivekit/go/modules/vcache/pkg"
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
		Constructor: discoverypkg.NewDiscoveryClientFactory,
	},
	{
		Type:        discovery.DirectoryDiscoveryServerModuleType,
		Constructor: discoverypkg.NewDirectoryDiscoveryServerFactory,
	},
	{
		Type:        discovery.ThingDiscoveryServerModuleType,
		Constructor: discoverypkg.NewThingDiscoveryServerFactory,
	},
	// gRPC transport
	{
		Type:        grpctransport.HiveotGrpcClientModuleType,
		Constructor: grpcpkg.NewHiveotGrpcClientFactory,
	},
	{
		Type:        grpctransport.HiveotGrpcServerModuleType,
		Constructor: grpcpkg.NewHiveotGrpcServerFactory,
	},
	// http server provider
	{
		Type:        api.HttpServerModuleType,
		Constructor: tlsserverpkg.NewTLSServerFactory,
	},
	// http-basic transport
	{
		Type:        httpbasic.HttpBasicClientModuleType,
		Constructor: httpbasicpkg.NewHttpBasicClientFactory,
	},
	{
		Type:        httpbasic.HttpBasicServerModuleType,
		Constructor: httpbasicpkg.NewHttpBasicServerFactory,
	},
	// sse-sc transport
	{
		Type:        ssesc.SseScServerModuleType,
		Constructor: ssescpkg.NewSseScServerFactory,
	},
	{
		Type:        ssesc.SseScClientModuleType,
		Constructor: ssescpkg.NewSseScClientFactory,
	},
	// wss transport for hiveot RRN messaging
	{
		Type:        wss.HiveotWebsocketClientModuleType,
		Constructor: wsspkg.NewHiveotWssClientFactory,
	},
	{
		Type:        wss.HiveotWebsocketServerModuleType,
		Constructor: wsspkg.NewHiveotWssServerFactory,
	},
	// wss transport for WoT websocket messaging
	{
		Type:        wss.WotWebsocketClientModuleType,
		Constructor: wsspkg.NewWotWssClientFactory,
	},
	{
		Type:        wss.WotWebsocketServerModuleType,
		Constructor: wsspkg.NewWotWssServerFactory,
	},

	//--- services ---

	// add forms to createTD or updateTD requests
	{
		Type:        addforms.AddFormsModuleType,
		Constructor: addformspkg.NewAddFormsServiceFactory,
	},

	// thing service helper
	{
		Type:        thing.ExposedThingModuleType,
		Constructor: thing.NewExposedThingFactory,
	},

	// client and session management provider
	{
		Type:        authnapi.AuthnServiceModuleType,
		Constructor: authnpkg.NewAuthnServiceFactory,
	},
	// authorization provider
	{
		Type:        authz.AuthzServiceModuleType,
		Constructor: authzpkg.NewAuthzServiceFactory,
	},
	// bucket store as a service
	{
		Type:        bucketstore.BucketStoreModuleType,
		Constructor: bucketstorepkg.NewBucketStoreServiceFactory,
	},
	// certs service
	{
		Type:        certs.CertsServerModuleType,
		Constructor: certspkg.NewCertsServiceFactory,
	},
	// InitFactoryCerts ensure the factory has certificates needed to run.
	{
		Type:        certs.InitFactoryCertsModuleType,
		Constructor: certspkg.NewInitFactoryCerts,
	},
	// consumer helper
	{
		Type:        consumer.ConsumerModuleType,
		Constructor: consumer.NewConsumerFactory,
	},

	// digitwin service
	{
		Type:        digitwin.DigitwinModuleType,
		Constructor: digitwinpkg.NewDigitwinServiceFactory,
	},
	// directory service
	{
		Type:        directory.DirectoryServiceModuleType,
		Constructor: directorypkg.NewDirectoryServiceFactory,
	},
	{
		Type:        directory.DirectoryClientModuleType,
		Constructor: directorypkg.NewDirectoryClientFactory,
	},
	// history service provider
	{
		Type:        history.HistoryModuleType,
		Constructor: historypkg.NewHistoryServiceFactory,
	},
	{
		Type:        history.ReadHistoryClientModuleType,
		Constructor: historypkg.NewReadHistoryClientFactory,
	},
	// logging service provider
	{
		Type:        logging.LoggingServiceModuleType,
		Constructor: loggingpkg.NewLoggingServiceFactory,
	},
	// auto-reconnect client
	{
		Type:        reconnect.ReconnectModuleType,
		Constructor: reconnectpkg.NewReconnectFactory,
	},
	// router service provider
	{
		Type:        router.RouterModuleType,
		Constructor: routerpkg.NewRouterServiceFactory,
	},
	// vcache server provider
	{
		Type:        vcache.ValueCacheModuleType,
		Constructor: vcachepkg.NewValueCacheServiceFactory,
	},
}
