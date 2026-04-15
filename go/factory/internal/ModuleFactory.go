package internal

import (
	"fmt"
	"log/slog"
	"sync"

	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// ModuleFactory for creating instances of modules using the application environment.
type ModuleFactory struct {
	env *factoryapi.AppEnvironment

	// when connecting a client interface using NewModuleClient
	// clientID string
	// authentication token ...?
	// authToken string
	// the root directory of the configuration storage directory
	// configRoot string

	// the root directory of the application storage area (subdir per module)
	// storageRoot string
	// the default timeout for transport modules
	// timeout time.Duration

	// the http server with and modules that serve http endpoints
	httpServer transports.IHttpServer

	// the module definition table, used for creating module instances by name
	moduleTable map[string]factoryapi.ModuleDefinition

	// instances of modules marked as singleton
	singletonModules map[string]modules.IHiveModule

	// the authenticator proxy
	authProxy *AuthenticatorProxy

	mux sync.RWMutex
}

// Return the application environment used by the factory.
func (f *ModuleFactory) GetEnvironment() *factoryapi.AppEnvironment {
	return f.env
}

// Used for server modules that need to authenticate incoming connections
// This returns a proxy to the actual authenticator.
func (f *ModuleFactory) GetAuthenticator() transports.IAuthenticator {
	return f.authProxy
}

// Used for various modules that need to serve http endpoints, e.g. http basic authn, directory, etc.
func (f *ModuleFactory) GetHttpServer() transports.IHttpServer {
	f.mux.RLock()
	httpServer := f.httpServer
	f.mux.RUnlock()

	if httpServer != nil {
		return httpServer
	}

	m, err := f.GetModule(transports.HttpServerModuleType)
	if err != nil {
		slog.Warn("GetHttpServer: no http server module is registered")
		return nil
	}
	httpServer, ok := m.(transports.IHttpServer)
	if !ok {
		slog.Error("The http server module does not support the IHttpServer API")
	}
	f.mux.Lock()
	f.httpServer = httpServer
	f.mux.Unlock()
	return httpServer
}

// GetModule loads and starts an instance of a module by its type.
// If the module is a singleton then the same instance is returned for multiple calls with the same type.
func (f *ModuleFactory) GetModule(moduleType string) (modules.IHiveModule, error) {
	m, isNew, err := f.LoadModule(moduleType)
	if err != nil {
		return nil, err
	}
	if isNew {
		err = m.Start()
	}
	return m, err
}

// LoadModule loads an instance of a module but does not start it yet.
// If the module is a singleton then the same instance is returned for multiple calls with the same type.
//
// The caller has to start the module before it can be used.
func (f *ModuleFactory) LoadModule(moduleType string) (m modules.IHiveModule, isNew bool, err error) {
	f.mux.RLock()
	m, ok := f.singletonModules[moduleType]
	f.mux.RUnlock()
	if m != nil {
		return m, false, nil
	}

	def, ok := f.moduleTable[moduleType]
	if !ok {
		err := fmt.Errorf("LoadModule: module '%s' not found", moduleType)
		slog.Error(err.Error())
		return nil, false, err
	}
	slog.Info("LoadModule loaded new module instance", "moduleType", moduleType)
	constructor := def.Constructor
	mod := constructor(f)

	// store the singleton on successful start
	if def.Singleton {
		f.mux.Lock()
		f.singletonModules[moduleType] = mod
		f.mux.Unlock()
	}
	return mod, true, nil
}

// RegisterModule registers a module definition to the factory, making it available for creation.
func (f *ModuleFactory) RegisterModule(moduleType string, moduleDef factoryapi.ModuleDefinition) {
	f.mux.Lock()
	defer f.mux.Unlock()
	f.moduleTable[moduleType] = moduleDef
}

func (f *ModuleFactory) SetAuthenticator(impl transports.IAuthenticator) {
	f.authProxy.SetAuthenticator(impl)
}

// NewModuleClient returns a client instance of a module with the given name.
// If the name is unknown this returns nil.
// func (f *ModuleFactory) NewModuleClient(name string) (m modules.IHiveModule) {

// 	switch name {
// 	case DirectoryClient:
// 		m = directory_api.NewDirectoryMsgClient("")
// 	}
// 	return m
// }

// Create a new instance of the module factory
//
// clientID is the client ID the modules created with this factory identify as.
// token is the auth token for use by the modules
// caCert is the CA used for clients and servers
// timeout is the connection timeout for use by clients
// func NewModuleFactory(clientID string, token string, caCert *x509.Certificate, timeout time.Duration) *ModuleFactory {
// 	f := ModuleFactory{
// 		clientID:   clientID,
// 		authToken:  token,
// 		caCert:     caCert,
// 		timeout:    timeout,
// 		httpServer: nil,
// 	}
// 	return &f
// }

// NewModuleFactory creates a new instance of the module factory using the application
// environment and an optional module definition table.
//
// If moduleTable is nil then an empty table is used, and modules can be added using AddModule.
func NewModuleFactory(
	env *factoryapi.AppEnvironment, moduleTable map[string]factoryapi.ModuleDefinition) factoryapi.IModuleFactory {

	if moduleTable == nil {
		moduleTable = make(map[string]factoryapi.ModuleDefinition)
	}
	f := &ModuleFactory{
		authProxy:        NewAuthenticatorProxy(),
		env:              env,
		moduleTable:      moduleTable,
		singletonModules: make(map[string]modules.IHiveModule),
	}
	var _ factoryapi.IModuleFactory = f // API check
	return f
}
