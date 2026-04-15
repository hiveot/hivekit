package factory_test

import (
	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules/authn"
	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"
	"github.com/hiveot/hivekit/go/modules/authz"
	authzapi "github.com/hiveot/hivekit/go/modules/authz/api"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketstoreapi "github.com/hiveot/hivekit/go/modules/bucketstore/api"
	"github.com/hiveot/hivekit/go/modules/certs"
	certsapi "github.com/hiveot/hivekit/go/modules/certs/api"
	"github.com/hiveot/hivekit/go/modules/digitwin"
	digitwinapi "github.com/hiveot/hivekit/go/modules/digitwin/api"
	"github.com/hiveot/hivekit/go/modules/directory"
	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
	"github.com/hiveot/hivekit/go/modules/history"
	historyapi "github.com/hiveot/hivekit/go/modules/history/api"
	"github.com/hiveot/hivekit/go/modules/logging"
	loggingapi "github.com/hiveot/hivekit/go/modules/logging/api"
	"github.com/hiveot/hivekit/go/modules/router"
	routerapi "github.com/hiveot/hivekit/go/modules/router/api"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/discovery"
	discoveryapi "github.com/hiveot/hivekit/go/modules/transports/discovery/api"
	grpctransport "github.com/hiveot/hivekit/go/modules/transports/grpc"
	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic"
	httpbasicapi "github.com/hiveot/hivekit/go/modules/transports/httpbasic/api"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	"github.com/hiveot/hivekit/go/modules/transports/ssesc"
	ssescapi "github.com/hiveot/hivekit/go/modules/transports/ssesc/api"
	wsstransport "github.com/hiveot/hivekit/go/modules/transports/wss"
	wssapi "github.com/hiveot/hivekit/go/modules/transports/wss/api"
	"github.com/hiveot/hivekit/go/modules/vcache"
	vcacheapi "github.com/hiveot/hivekit/go/modules/vcache/api"
)

// Table of modules used for running servers.
var ServerModuleTable = map[string]factoryapi.ModuleDefinition{

	//--- transport servers ---

	// discovery transport server provider
	discoveryapi.DiscoveryServerModuleType: {
		Singleton:   true,
		Constructor: discovery.NewDiscoveryServerFactory,
	},
	// gRPC transport server
	grpcapi.HiveotGrpcModuleType: {
		Singleton:   true,
		Constructor: grpctransport.NewHiveotGrpcServerFactory,
	},
	// http server provider
	transports.HttpServerModuleType: {
		Singleton:   true,
		Constructor: httpserver.NewHttpServerFactory,
	},
	// http-basic transport server
	httpbasicapi.HttpBasicServerModuleType: {
		Singleton:   true,
		Constructor: httpbasic.NewHttpBasicServerFactory,
	},
	// sse-sc transport server
	ssescapi.SseScServerModuleType: {
		Singleton:   true,
		Constructor: ssesc.NewSseScServerFactory,
	},
	// wss transport server for hiveot RRN messaging
	wssapi.HiveotWebsocketModuleType: {
		Singleton:   true,
		Constructor: wsstransport.NewHiveotWssServerFactory,
	},
	// wss transport server for WoT websocket messaging
	wssapi.WotWebsocketModuleType: {
		Singleton:   true,
		Constructor: wsstransport.NewWotWssServerFactory,
	},

	//--- services servers ---

	// client and session management provider
	authnapi.AuthnModuleType: {
		Singleton:   true,
		Constructor: authn.NewAuthnServiceFactory,
	},
	// authorization provider
	authzapi.AuthzModuleType: {
		Singleton:   true,
		Constructor: authz.NewAuthzServiceFactory,
	},
	// bucket store as a service
	bucketstoreapi.BucketStoreModuleType: {
		Singleton:   true,
		Constructor: bucketstore.NewBucketStoreServiceFactory,
	},
	// certs service provider
	certsapi.CertsModuleType: {
		Singleton:   true,
		Constructor: certs.NewCertsServiceFactory,
	},
	// digitwin provider
	digitwinapi.DigitwinModuleType: {
		Singleton:   true,
		Constructor: digitwin.NewDigitwinServiceFactory,
	},
	// directory service provider
	directoryapi.DirectoryModuleType: {
		Singleton:   true,
		Constructor: directory.NewDirectoryServiceFactory,
	},
	// history service provider
	historyapi.HistoryModuleType: {
		Singleton:   true,
		Constructor: history.NewHistoryServiceFactory,
	},
	// logging service provider
	loggingapi.LoggingModuleType: {
		Singleton:   true,
		Constructor: logging.NewLoggingServiceFactory,
	},
	// router service provider
	routerapi.RouterModuleType: {
		Singleton:   true,
		Constructor: router.NewRouterServiceFactory,
	},
	// vcache server provider
	vcacheapi.VCacheModuleType: {
		Singleton:   true,
		Constructor: vcache.NewVCacheServiceFactory,
	},
}
