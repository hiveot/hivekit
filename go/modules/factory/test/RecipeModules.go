package factory_test

import (
	"github.com/hiveot/hivekit/go/modules/agent"
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
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/history"
	historypkg "github.com/hiveot/hivekit/go/modules/history/pkg"
	"github.com/hiveot/hivekit/go/modules/logging"
	loggingpkg "github.com/hiveot/hivekit/go/modules/logging/pkg"
	"github.com/hiveot/hivekit/go/modules/reconnect"
	reconnectpkg "github.com/hiveot/hivekit/go/modules/reconnect/pkg"
	"github.com/hiveot/hivekit/go/modules/router"
	routerpkg "github.com/hiveot/hivekit/go/modules/router/pkg"
	"github.com/hiveot/hivekit/go/modules/transport"
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
	vcacheapi "github.com/hiveot/hivekit/go/modules/vcache/api"
)

// List hivekit available modules
var HiveKitModules = []factory.ModuleDefinition{

	//--- transport servers ---

	// discovery transport server provider
	{
		Type:        discovery.DiscoveryServerModuleType,
		Constructor: discoverypkg.NewDiscoveryServerFactory,
	},
	// gRPC transport server
	{
		Type:        grpctransport.HiveotGrpcServerModuleType,
		Constructor: grpcpkg.NewGrpcServerFactory,
	},
	// http server provider
	{
		Type:        transport.TLSServerModuleType,
		Constructor: tlsserverpkg.NewTLSServerFactory,
	},
	// http-basic transport server
	{
		Type:        httpbasic.HttpBasicServerModuleType,
		Constructor: httpbasicpkg.NewHttpBasicServerFactory,
	},
	// sse-sc transport server
	{
		Type:        ssesc.SseScServerModuleType,
		Constructor: ssescpkg.NewSseScServerFactory,
	},
	// wss transport server for hiveot RRN messaging
	{
		Type:        wss.HiveotWebsocketServerModuleType,
		Constructor: wsspkg.NewHiveotWssServerFactory,
	},
	// wss transport server for WoT websocket messaging
	{
		Type:        wss.WotWebsocketServerModuleType,
		Constructor: wsspkg.NewWotWssServerFactory,
	},

	// clients
	{
		Type:        agent.AgentModuleType,
		Constructor: agent.NewAgentFactory,
	},
	{
		Type:        consumer.ConsumerModuleType,
		Constructor: consumer.NewConsumerFactory,
	},
	{
		Type:        reconnect.ReconnectModuleType,
		Constructor: reconnectpkg.NewReconnectFactory,
	},
	{
		Type:        history.ReadHistoryClientModuleType,
		Constructor: historypkg.NewReadHistoryClientFactory,
	},
	// sse-sc transport client
	{
		Type:        ssesc.SseScClientModuleType,
		Constructor: ssescpkg.NewSseScClientFactory,
	},
	// wss transport client for hiveot RRN messaging
	{
		Type:        wss.HiveotWebsocketClientModuleType,
		Constructor: wsspkg.NewHiveotWssClientFactory,
	},
	// wss transport client for WoT websocket messaging
	{
		Type:        wss.WotWebsocketClientModuleType,
		Constructor: wsspkg.NewWotWssClientFactory,
	},

	//--- services servers ---

	// add forms to createTD or updateTD requests
	{
		Type:        addforms.AddFormsModuleType,
		Constructor: addformspkg.NewAddFormsServiceFactory,
	},

	// client and session management provider
	{
		Type:        authnapi.AuthnServiceModuleType,
		Constructor: authnpkg.NewAuthnServiceFactory,
	},
	// authorization provider
	{
		Type:        authz.AuthzModuleType,
		Constructor: authzpkg.NewAuthzServiceFactory,
	},
	// bucket store as a service
	{
		Type:        bucketstore.BucketStoreModuleType,
		Constructor: bucketstorepkg.NewBucketStoreServiceFactory,
	},
	// certs service provider
	{
		Type:        certs.CertsServerModuleType,
		Constructor: certspkg.NewCertsServiceFactory,
	},
	// InitFactoryCerts ensure the factory has certificates needed to run.
	{
		Type:        certs.InitFactoryCertsModuleType,
		Constructor: certspkg.NewInitFactoryCerts,
	},
	// digitwin provider
	{
		Type:        digitwin.DigitwinModuleType,
		Constructor: digitwinpkg.NewDigitwinServiceFactory,
	},
	// directory service provider
	{
		Type:        directory.DirectoryModuleType,
		Constructor: directorypkg.NewDirectoryServiceFactory,
	},
	// history service provider
	{
		Type:        history.HistoryModuleType,
		Constructor: historypkg.NewHistoryServiceFactory,
	},
	// logging service provider
	{
		Type:        logging.LoggingModuleType,
		Constructor: loggingpkg.NewLoggingServiceFactory,
	},
	// router service provider
	{
		Type:        router.RouterModuleType,
		Constructor: routerpkg.NewRouterServiceFactory,
	},
	// vcache server provider
	{
		Type:        vcacheapi.VCacheModuleType,
		Constructor: vcache.NewVCacheServiceFactory,
	},
}
