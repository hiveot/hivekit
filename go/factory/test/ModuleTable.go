package factory_test

import (
	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules/authn"
	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"
	"github.com/hiveot/hivekit/go/modules/authz"
	authzapi "github.com/hiveot/hivekit/go/modules/authz/api"
	"github.com/hiveot/hivekit/go/modules/directory"
	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
)

// Table of modules used for running servers.
var ServerModuleTable = map[string]factoryapi.ModuleDefinition{
	// transport authenticator provider
	transports.AuthenticatorModuleType: {
		Singleton:   true,
		Constructor: authn.NewAuthenticatorFactory,
	},
	// authentication management provider
	authnapi.AuthnModuleType: {
		Singleton:   true,
		Constructor: authn.NewAuthnServiceFactory,
	},
	// authorization provider
	authzapi.AuthzModuleType: {
		Singleton:   true,
		Constructor: authz.NewAuthzServiceFactory,
	},
	// directory service provider
	directoryapi.DirectoryModuleType: {
		Singleton:   true,
		Constructor: directory.NewDirectoryServiceFactory,
	},
	// http server provider
	transports.HttpServerModuleType: {
		Singleton:   true,
		Constructor: httpserver.NewHttpServerFactory,
	},
}
