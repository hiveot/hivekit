package factory_test

import (
	authnapi "github.com/hiveot/hivekit/go/modules/authn"
	authnpkg "github.com/hiveot/hivekit/go/modules/authn/pkg"
	"github.com/hiveot/hivekit/go/modules/authz"
	authzpkg "github.com/hiveot/hivekit/go/modules/authz/pkg"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketstorepkg "github.com/hiveot/hivekit/go/modules/bucketstore/pkg"
	"github.com/hiveot/hivekit/go/modules/certs"
	certspkg "github.com/hiveot/hivekit/go/modules/certs/pkg"
	clientspkg "github.com/hiveot/hivekit/go/modules/clients/pkg"
	"github.com/hiveot/hivekit/go/modules/digitwin"
	digitwinpkg "github.com/hiveot/hivekit/go/modules/digitwin/pkg"
	"github.com/hiveot/hivekit/go/modules/directory"
	directorypkg "github.com/hiveot/hivekit/go/modules/directory/pkg"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/history"
	historypkg "github.com/hiveot/hivekit/go/modules/history/pkg"
	"github.com/hiveot/hivekit/go/modules/logging"
	loggingpkg "github.com/hiveot/hivekit/go/modules/logging/pkg"
	"github.com/hiveot/hivekit/go/modules/router"
	routerpkg "github.com/hiveot/hivekit/go/modules/router/pkg"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/addforms"
	addformspkg "github.com/hiveot/hivekit/go/modules/transports/addforms/pkg"
	"github.com/hiveot/hivekit/go/modules/transports/discovery"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transports/discovery/pkg"
	grpctransport "github.com/hiveot/hivekit/go/modules/transports/grpc"
	grpcpkg "github.com/hiveot/hivekit/go/modules/transports/grpc/pkg"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic"
	httpbasicpkg "github.com/hiveot/hivekit/go/modules/transports/httpbasic/pkg"
	httptransportpkg "github.com/hiveot/hivekit/go/modules/transports/httptransport/pkg"
	"github.com/hiveot/hivekit/go/modules/transports/ssesc"
	ssescpkg "github.com/hiveot/hivekit/go/modules/transports/ssesc/pkg"
	wss "github.com/hiveot/hivekit/go/modules/transports/wss"
	wsspkg "github.com/hiveot/hivekit/go/modules/transports/wss/pkg"

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
	transports.HttpServerModuleType: {
		Constructor: httptransportpkg.NewHttpTransportServerFactory,
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

	//--- services servers ---

	// add forms to createTD or updateTD requests
	addforms.AddFormsModuleType: {
		Constructor: addformspkg.NewAddFormsServiceFactory,
	},

	// client and session management provider
	authnapi.AuthnModuleType: {
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
		Constructor: directorypkg.NewDirectoryMsgServerFactory,
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

	// clients
	clientspkg.AgentModuleType: {
		Constructor: clientspkg.NewAgentFactory,
	},
	clientspkg.ConsumerModuleType: {
		Constructor: clientspkg.NewConsumerFactory,
	},
	history.ReadHistoryClientModuleType: {
		Constructor: historypkg.NewReadHistoryClientFactory,
	},
}
