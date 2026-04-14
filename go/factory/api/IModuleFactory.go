package factoryapi

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// the constructor function to create an instance of the module using the given environment
// The recommended moduleID is auto-generated. The module can decide to override if needed.
type ModuleFactoryFn func(f IModuleFactory) modules.IHiveModule

// ModuleDefinition defines the constructor for a module, used for registration in the module factory
// This can also be used to add custom modules.
type ModuleDefinition struct {

	// Set singleton to true to create only one instance of the module.
	// Multiple instances of the module require different instance IDs.
	Singleton bool

	// Type of the module, used for registration and lookup.
	// Note that the module type is identical for all instances of a module and is used in the @type
	// field of the module TM, if used. The moduleID is the instance ID of the module and
	// must be unique. Singleton modules use the same ID for module type and moduleID.
	Type string

	// the constructor function to create an instance of the module
	Constructor ModuleFactoryFn
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

	// Provide the means to authenticate incoming connections
	// Intended for transport server modules.
	// This requires that an authn module is registered.
	GetAuthenticator() transports.IAuthenticator

	// GetEnvironment returns the application environment used by the factory for
	// confuring modules.
	GetEnvironment() *AppEnvironment

	// Used for modules that need to serve http endpoints, e.g. http basic authn, directory, etc.
	// This requires that an httpserver module is registered.
	GetHttpServer() transports.IHttpServer

	// GetModule creates and starts an instance of a module by its type.
	//
	// If the module is defined as a singleton, the existing module is returned.
	// If no module instance exist or it isn't a singleton then create a new instance.
	//
	//  moduleType identifies the type of the module to create.
	//
	// This returns an error if no module with the given type is registered, or when
	// starting the module fails.
	GetModule(moduleType string) (modules.IHiveModule, error)

	// RegisterModule a module to the factory, making it available for creation.
	//
	// moduleType identifies the type of the module. (not instance)
	// moduleDef defines the module attributes and constructor function
	RegisterModule(moduleType string, moduleDef ModuleDefinition)
}
