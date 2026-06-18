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

// Map of available modules
var RecipeModules = map[string]factory.ModuleDefinition{

	//--- transport servers ---

	// discovery transport server provider
	discovery.DiscoveryServerModuleType: {
		Constructor: discoverypkg.NewDiscoveryServerFactory,
	},
	// gRPC transport server
	grpctransport.HiveotGrpcServerModuleType: {
		Constructor: grpcpkg.NewGrpcServerFactory,
	},
	// http server provider
	transport.TLSServerModuleType: {
		Constructor: tlsserverpkg.NewTLSServerFactory,
	},
	// http-basic transport server
	httpbasic.HttpBasicServerModuleType: {
		Constructor: httpbasicpkg.NewHttpBasicServerFactory,
	},
	// sse-sc transport client
	ssesc.SseScClientModuleType: {
		Constructor: ssescpkg.NewSseScClientFactory,
	},
	// sse-sc transport server
	ssesc.SseScServerModuleType: {
		Constructor: ssescpkg.NewSseScServerFactory,
	},
	// wss transport client for hiveot RRN messaging
	wss.HiveotWebsocketClientModuleType: {
		Constructor: wsspkg.NewHiveotWssClientFactory,
	},
	// wss transport server for hiveot RRN messaging
	wss.HiveotWebsocketServerModuleType: {
		Constructor: wsspkg.NewHiveotWssServerFactory,
	},
	// wss transport client for WoT websocket messaging
	wss.WotWebsocketClientModuleType: {
		Constructor: wsspkg.NewWotWssClientFactory,
	},
	// wss transport server for WoT websocket messaging
	wss.WotWebsocketServerModuleType: {
		Constructor: wsspkg.NewWotWssServerFactory,
	},

	// clients
	agent.AgentModuleType: {
		Constructor: agent.NewAgentFactory,
	},
	consumer.ConsumerModuleType: {
		Constructor: consumer.NewConsumerFactory,
	},
	reconnect.ReconnectModuleType: {
		Constructor: reconnectpkg.NewReconnectFactory,
	},
	history.ReadHistoryClientModuleType: {
		Constructor: historypkg.NewReadHistoryClientFactory,
	},

	//--- services servers ---

	// add forms to createTD or updateTD requests
	addforms.AddFormsModuleType: {
		Constructor: addformspkg.NewAddFormsServiceFactory,
	},

	// client and session management provider
	authnapi.AuthnServiceModuleType: {
		Constructor: authnpkg.NewAuthnServiceFactory,
	},
	// authorization provider
	authz.AuthzModuleType: {
		Constructor: authzpkg.NewAuthzServiceFactory,
	},
	// bucket store as a service
	bucketstore.BucketStoreModuleType: {
		Constructor: bucketstorepkg.NewBucketStoreServiceFactory,
	},
	// certs service provider
	certs.CertsServerModuleType: {
		Constructor: certspkg.NewCertsServiceFactory,
	},
	// InitFactoryCerts ensure the factory has certificates needed to run.
	certs.InitFactoryCertsModuleType: {
		Constructor: certspkg.NewInitFactoryCerts,
	},
	// digitwin provider
	digitwin.DigitwinModuleType: {
		Constructor: digitwinpkg.NewDigitwinServiceFactory,
	},
	// directory service provider
	directory.DirectoryModuleType: {
		Constructor: directorypkg.NewDirectoryServiceFactory,
	},
	// history service provider
	history.HistoryModuleType: {
		Constructor: historypkg.NewHistoryServiceFactory,
	},
	// logging service provider
	logging.LoggingModuleType: {
		Constructor: loggingpkg.NewLoggingServiceFactory,
	},
	// router service provider
	router.RouterModuleType: {
		Constructor: routerpkg.NewRouterServiceFactory,
	},
	// vcache server provider
	vcacheapi.VCacheModuleType: {
		Constructor: vcache.NewVCacheServiceFactory,
	},
}
