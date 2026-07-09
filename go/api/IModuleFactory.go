package api

import (
	"context"

	"github.com/hiveot/hivekit/go/api/td"
)

// the constructor function to create an instance of the module using the given environment
// The recommended moduleID is auto-generated. The module can decide to override if needed.
// type ModuleFactoryFn func(f IModuleFactory) api.IHiveModule

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

	// The constructor function to create an instance of the module.
	// The configuration can be used to pass arguments and other configuration to the module.
	//
	// f is the module factory with the app environment and ability to retrieve other modules
	// modDef is the module definition passed to the constructor.
	//
	// This returns an error if the module cannot be created.
	// This returns nil with no error for modules that are used for initialization.
	Constructor func(f IModuleFactory, modDef *ModuleDefinition) (IHiveModule, error)

	// Optional configuration passed to the creation of the module
	Config any
}

// IModuleFactory is the interface for the module factory, used to create and manage
// modules by their type.
//
// The module factory can be used stand-alone or together with the ChainRecipe or StarRecipe.
type IModuleFactory interface {

	// Add security and forms to the TD for all running transport protocols
	// Intended for devices to add forms before exporting a TD.
	// This passes the request to all server instances that have been created using
	// this factory.
	AddTDSecForms(tdoc *td.TD, includeAffordances bool)

	// Provide the means to authenticate incoming connections.
	// Intended for transport server modules.
	// This returns a proxy stub that can be updated with SetAuthenticator.
	// If no authenticator is set the this proxy fails all authentication attempts.
	//
	// SetAuthenticator is called by the authn module when it is created.
	GetAuthenticator() IAuthenticator

	// Get the connection URL of the first loaded server module or "" if none.
	// Primarily intended for testing. It is recommended to use a discovery server/client module
	// in the factory server/client chains to facilitate discovery of server by the client.
	GetConnectURL() string

	// GetEnvironment returns the application environment used by the factory for
	// confuring modules.
	// Note that the environment can be updated by the modules to allow factory modules
	// to update the TDD, location of gateway and other discoverable information.
	GetEnvironment() *AppEnvironment

	// GetModule returns the loaded module of the given module type.
	//
	// If a module hasn't been loaded/started yet then this returns nil.
	GetModule(moduleType string) IHiveModule

	// Return the http server module instance.
	//
	// Used for modules that need to serve http endpoints, e.g. http basic authn, directory, etc.
	//
	// Set the instantiate flag to indicate that the http server module of type TLSServerModuleType
	// should be loaded if it hasn't been loaded yet. If no such module is registered in the factory
	// module definitions then this returns nil and a warning is logged.
	//
	//  instantiate set to true to auto load the http server module
	//
	// This returns nil if no httpserver module is registered.
	GetHttpServer(instantiate bool) IHttpServer

	// Obtain the directory TD.
	// Intended for bootstrapping the directory client.
	// GetTDD() *td.TD

	// Return the list of available transport servers
	GetTransportServers() []ITransportServer

	// RegisterModule adds a module to the factory, making it available for instantiation
	// and for running recipes.
	//
	// If a module is already registered it is replaced. If the given definition
	// doesn't contain a factory constructor but the existing registration does then
	// only the config from the definition is used and merged with the existing registration.
	//
	// Intended to allow pre-registering modules and only include a ordered list of
	// modules in the chain to instantiate and link.
	//
	// moduleDef defines the module attributes and constructor function
	RegisterModule(moduleDef ModuleDefinition)

	// SetAuthenticator sets the authenticator returned by GetAuthenticator.
	// Note that GetAuthenticator returns a proxy to the actual authenticator.
	// Intended for use by the module that offers authentication capabilities,
	// such as the authn module.
	//
	// By default the authenticator proxy blocks all authentication.
	// Setting a nil authenticator disables authentication.
	SetAuthenticator(a IAuthenticator)

	// StartModule creates and starts an instance of a module by its type.
	//
	// If the module is already started, the existing module instance is returned.
	//
	// If the module factory function is nil then this is an empty slot which
	// will be ignored.
	//
	// This does not link the module to other modules. See also RunRecipe for creating a chain.
	//
	//  moduleType identifies the type of the module to get.
	//	instantiate set to true to create an instance if one isnt loaded
	//
	// This returns an error if no module with the given type is found, or when
	// starting the module fails.
	// This returns nil with no error if the module factory is a 'one-shot'
	// initialization function where its factory handler returns nil.
	StartModule(moduleType string, instantiate bool) (IHiveModule, error)

	// Stop all loaded modules in reverse order of loading.
	// Intended for graceful shutdown.
	Stop()

	// WaitForSignal waits until an OS SigTerm signal is received or context is cancelled.
	// Call StopAll() afters this returns for proper cleanup.
	WaitForSignal(ctx context.Context)
}

// Helper to get a module from the factory with the given interface
func GetFactoryModule[T interface{}](f IModuleFactory, moduleType string) T {
	m := f.GetModule(moduleType)
	t, _ := m.(T)
	return t
}
