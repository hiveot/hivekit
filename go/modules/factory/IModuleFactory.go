package factory

import (
	"context"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// the constructor function to create an instance of the module using the given environment
// The recommended moduleID is auto-generated. The module can decide to override if needed.
// type ModuleFactoryFn func(f IModuleFactory) modules.IHiveModule

// ModuleDefinition defines the constructor for a module, used for registration in the module factory
// This can also be used to add custom modules.
type ModuleDefinition struct {

	// Set Multiton to true to allow multiple instances of the module.
	// Multiton instances of the module require different instance IDs.
	// Multiton bool

	// Type of the module, used for registration and lookup.
	// Note that the module type is identical for all instances of a module and is used in the @type
	// field of the module TM, if used. The moduleID is the instance ID of the module and
	// must be unique. Singleton modules use the same ID for module type and moduleID.
	Type string

	// the constructor function to create an instance of the module
	Constructor func(f IModuleFactory) modules.IHiveModule
}

// IModuleFactory is the interface for the module factory, used to create and manage
// modules by their type.
//
// Since clients of services are also modules this can be used to create both the client and
// server/service modules. These are registered separately.
//
// Modules that contain multiple services (like authn admin vs user) can also register
// as a single module and operate multiple thingID's.
// This is common for protocol bindings like zwave or other device protocols where the module
// manages multiple things.
type IModuleFactory interface {
	modules.IHiveModule // the factory is also a module

	// Add security and forms to the TD for all running transport protocols
	// Intended for devices to add forms before exporting a TD.
	AddTDSecForms(tdoc *td.TD, includeAffordances bool)

	// FindModule returns the loaded module of the given type
	// This returns nil if no such module is loaded
	FindModule(moduleType string) modules.IHiveModule

	// Provide the means to authenticate incoming connections.
	// Intended for transport server modules.
	// This returns a proxy stub that can be updated with SetAuthenticator.
	// If no authenticator is set the this proxy fails all authentication attempts.
	//
	// SetAuthenticator is called by the authn module when it is created.
	GetAuthenticator() transports.IAuthenticator

	// Get the connection URL of the first loaded server module or "" if none.
	// Primarily intended for testing. It is recommended to use a discovery module in the
	// factory server and client chains to facilitate discovery of server by the client.
	GetConnectURL() string

	// GetEnvironment returns the application environment used by the factory for
	// confuring modules.
	GetEnvironment() *AppEnvironment

	// Return the http module if it was instantiated.
	// Used for modules that need to serve http endpoints, e.g. http basic authn, directory, etc.
	//
	//  instantiate true to create the server module instance if it hasn't been created yet
	//
	// This returns nil if no httpserver module is registered.
	GetHttpServer(instantiate bool) transports.IHttpServer

	// Return the list of available transport servers
	GetTransportServers() []transports.ITransportServer

	// GetModule creates and starts an instance of a module by its type.
	//
	// If the module is defined as a singleton, the existing module is returned.
	// If no module instance exist or it isn't a singleton then create a new instance.
	//
	//  moduleType identifies the type of the module to get.
	//	instantiate set to true to create an instance if one isn't already loaded
	//
	// This returns an error if no module with the given type is registered, when
	// starting the module fails.
	// This returns nil on error or when instantiate is false and the module is not yet loaded
	GetModule(moduleType string, instantiate bool) (modules.IHiveModule, error)

	// RegisterModule a module to the factory, making it available for creation.
	//
	// moduleType identifies the type of the module. (not instance)
	// moduleDef defines the module attributes and constructor function
	RegisterModule(moduleType string, moduleDef ModuleDefinition)

	// SetAuthenticator sets the authenticator returned by GetAuthenticator.
	// Note that GetAuthenticator returns a proxy to the actual authenticator.
	// Intended for use by the module that offers authentication capabilities,
	// such as the authn module. Other authentication modules can be used instead.
	//
	// NOTE: This should be deprecated. It is only here because the authentication module
	// cannot be loaded first as it exposes a http api and needs the http server.
	// TODO split this in 2 modules, one for authentication and another for
	// serving the http API.
	SetAuthenticator(a transports.IAuthenticator)

	// StartRecipe registers all modules in a recipe, start and link them in the prescribed order
	//
	// This returns a list of modules in the order they are loaded or an error when failed.
	//
	// If one module fails then an error is returned and all loaded modules will be stopped.
	// StartRecipe(recipe *FactoryRecipe) (modList []modules.IHiveModule, err error)

	// Stop all loaded modules in reverse order of loading.
	// Intended for graceful shutdown.
	StopAll()

	// WaitForSignal waits until an OS SigTerm signal is received or context is cancelled.
	// Call StopAll() afters this returns for proper cleanup.
	WaitForSignal(ctx context.Context)
}
