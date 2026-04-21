package internal

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// ModuleFactory for creating instances of modules using the application environment.
//
// This factory itself is the first module in the chain of modules created by this factory.
type ModuleFactory struct {
	modules.HiveModuleBase

	env *factory.AppEnvironment

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
	moduleTable map[string]factory.ModuleDefinition

	// list of loaded modules in order of instantiation
	loadedModules []modules.IHiveModule

	// instances of modules marked as singleton
	singletonModules map[string]modules.IHiveModule

	// list of all transport modules
	transportModules []transports.ITransportServer

	// the authenticator proxy
	authProxy *AuthenticatorProxy

	mux sync.RWMutex
}

// Add forms to the TD for all running transport servers
// This invokes all singletonModules that implement the ITransportServer interface
func (f *ModuleFactory) AddTDSecForms(tdoc *td.TD, includeAffordances bool) {
	f.mux.RLock()
	tpList := []transports.ITransportServer{}
	copy(tpList, f.transportModules)
	f.mux.RUnlock()
	for _, tp := range tpList {
		tp.AddTDSecForms(tdoc, includeAffordances)
	}
}

// Find the first loaded singleton module instance by its type
// This returns nil if no instance was loaded or the module isn't a singleton
func (f *ModuleFactory) FindModule(moduleType string) (m modules.IHiveModule) {
	f.mux.RLock()
	defer f.mux.RUnlock()
	m, ok := f.singletonModules[moduleType]
	_ = ok
	return m
}

// Return the application environment used by the factory.
func (f *ModuleFactory) GetEnvironment() *factory.AppEnvironment {
	return f.env
}

// Used for server modules that need to authenticate incoming connections
// This returns a proxy to the actual authenticator.
func (f *ModuleFactory) GetAuthenticator() transports.IAuthenticator {
	return f.authProxy
}

// Return the first loaded module. This returns nil if no modules are loaded
func (f *ModuleFactory) GetFirstModule() modules.IHiveModule {
	f.mux.RLock()
	defer f.mux.RUnlock()
	if len(f.loadedModules) > 0 {
		return f.loadedModules[0]
	}
	return nil
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

// Return the last loaded module. This returns nil if no modules are loaded
func (f *ModuleFactory) GetLastModule() modules.IHiveModule {
	f.mux.RLock()
	defer f.mux.RUnlock()
	if len(f.loadedModules) > 0 {
		return f.loadedModules[0]
	}
	return nil
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

// Return the connectURL of the first server
func (f *ModuleFactory) GetConnectURL() string {
	servers := f.GetTransportServers()
	if len(servers) == 0 {
		return ""
	}
	return servers[0].GetConnectURL()
}

// Return a copy of the list with loaded transport servers.
func (f *ModuleFactory) GetTransportServers() []transports.ITransportServer {
	f.mux.RLock()
	tpList := make([]transports.ITransportServer, len(f.transportModules))
	copy(tpList, f.transportModules)
	f.mux.RUnlock()
	return tpList
}

// LoadModule loads an instance of a module without starting it.
//
// If the module implements the ITransportModule interface it is added to the list of available
// transports. See GetTransportServers() to obtain the collection of all loaded servers.
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

	f.mux.Lock()
	f.singletonModules[moduleType] = mod
	tp, ok := mod.(transports.ITransportServer)
	if ok {
		f.transportModules = append(f.transportModules, tp)
	}
	f.mux.Unlock()

	// add to the loaded list
	f.mux.Lock()
	f.loadedModules = append(f.loadedModules, mod)
	f.mux.Unlock()
	return mod, true, nil
}

// RegisterModule registers a module definition to the factory, making it available for creation.
// Intended to support 3rd party modules.
func (f *ModuleFactory) RegisterModule(moduleType string, moduleDef factory.ModuleDefinition) {
	f.mux.Lock()
	defer f.mux.Unlock()
	f.moduleTable[moduleType] = moduleDef
}

func (f *ModuleFactory) SetAuthenticator(impl transports.IAuthenticator) {
	f.authProxy.SetAuthenticator(impl)
}

// Stop all modules in reverse order
func (f *ModuleFactory) StopAll() {
	n := len(f.loadedModules)
	slog.Info("StopAll: stopping all loaded modules", "count", n)
	for i := n - 1; i >= 0; i-- {
		m := f.loadedModules[i]
		m.Stop()
	}
	f.loadedModules = make([]modules.IHiveModule, 0)
}

// Wait for an OS signal or until the context is cancelled
func (f *ModuleFactory) WaitForSignal(ctx context.Context) {

	// catch all signals since not explicitly listing
	exitChannel := make(chan os.Signal, 1)

	signal.Notify(exitChannel, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sigID := <-exitChannel:
		println("WaitForSignal done with signal ", sigID, ": ", os.Args[0], "\n")
	case <-ctx.Done():
		println("WaitForSignal context closed")
	}
}

// Create a new module factory.
// Modules can be nil if they are registered separately or if StartRecipe is used.
//
//	env is the application enviroment created with factory.NewAppEnvironment
//	moduleDefs are the module definitions available to GetModule(type)
func NewModuleFactory(
	env *factory.AppEnvironment, moduleDefs map[string]factory.ModuleDefinition) factory.IModuleFactory {

	if moduleDefs == nil {
		moduleDefs = make(map[string]factory.ModuleDefinition)
	}
	f := &ModuleFactory{
		authProxy:        NewAuthenticatorProxy(),
		env:              env,
		moduleTable:      moduleDefs,
		singletonModules: make(map[string]modules.IHiveModule),
	}
	var _ factory.IModuleFactory = f // API check
	return f
}
