package internal

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
)

// FactoryImpl for creating instances of cells using the application environment.
type FactoryImpl struct {
	*cells.HiveCellBase

	env *api.HiveEnvironment

	// the http server with and cells that serve http endpoints
	httpServer api.IHttpServer

	// cell definitions used for creating cell instances by name
	cellDefinitions map[string]api.CellDefinition

	// list of loaded cells in order of instantiation
	loadedCells []api.IHiveCell

	// instances of cells marked as singleton
	singletonCells map[string]api.IHiveCell

	// list of all transport cells
	transportCells []api.ITransportServer

	// the authenticator proxy
	authProxy *AuthenticatorProxy

	mux sync.RWMutex
}

// Add forms to the TD for all running transport servers
// This invokes all singletonCells that implement the ITransportServer interface
func (f *FactoryImpl) AddTDSecForms(tdoc *td.TD, includeAffordances bool) {
	f.mux.RLock()
	tpList := []api.ITransportServer{}
	copy(tpList, f.transportCells)
	f.mux.RUnlock()
	for _, tp := range tpList {
		tp.AddTDSecForms(tdoc, includeAffordances)
	}
}

// Used for server cells that need to authenticate incoming connections
// This returns a proxy to the actual authenticator.
func (f *FactoryImpl) GetAuthenticator() api.IAuthenticator {
	return f.authProxy
}

// Return the application environment used by the factory.
func (f *FactoryImpl) GetEnvironment() *api.HiveEnvironment {
	return f.env
}

// GetCell returns the cell instance by its type
// This returns nil if no instance was loaded or the cell isn't a singleton
func (f *FactoryImpl) GetCell(cellType string) (m api.IHiveCell) {
	f.mux.RLock()
	defer f.mux.RUnlock()
	m, ok := f.singletonCells[cellType]
	_ = ok
	return m
}

// Return the first loaded cell. This returns nil if no cells are loaded
func (f *FactoryImpl) GetFirstCell() api.IHiveCell {
	f.mux.RLock()
	defer f.mux.RUnlock()
	if len(f.loadedCells) > 0 {
		return f.loadedCells[0]
	}
	return nil
}

// Used for various cells that need to serve http endpoints, e.g. http basic authn, directory, etc.
//
//	instantiate indicates if the http server instance should be created if it doesnt exist.
//
// This returns nil if no http server cell is registered
func (f *FactoryImpl) GetHttpServer(instantiate bool) api.IHttpServer {
	f.mux.RLock()
	httpServer := f.httpServer
	f.mux.RUnlock()

	if httpServer != nil {
		return httpServer
	}
	if !instantiate {
		return nil
	}
	m, err := f.StartCell(api.HttpServerCellType, instantiate)
	if err != nil {
		slog.Warn("GetHttpServer: no http server is registered")
		return nil
	}
	httpServer, ok := m.(api.IHttpServer)
	if !ok {
		slog.Error("The http server cell does not support the IHttpServer API")
	}
	f.mux.Lock()
	f.httpServer = httpServer
	f.mux.Unlock()
	return httpServer
}

// Return the last loaded cell. This returns nil if no cells are loaded
func (f *FactoryImpl) GetLastCell() api.IHiveCell {
	f.mux.RLock()
	defer f.mux.RUnlock()
	if len(f.loadedCells) > 0 {
		return f.loadedCells[0]
	}
	return nil
}

// Return the connectURL of the servers
func (f *FactoryImpl) GetConnectURLs() []string {
	servers := f.GetTransportServers()
	if len(servers) == 0 {
		return nil
	}
	urls := []string{}
	for _, srv := range servers {
		urls = append(urls, srv.GetConnectURL())
	}
	return urls
}

// Return a copy of the list with loaded transport servers.
// FIXME: this doesnt work for nested recipes? -> factory holds all cells so why not?
func (f *FactoryImpl) GetTransportServers() []api.ITransportServer {
	f.mux.RLock()
	tpList := make([]api.ITransportServer, len(f.transportCells))
	copy(tpList, f.transportCells)
	f.mux.RUnlock()
	return tpList
}

// Pass request to the first loaded cell in the factory
func (f *FactoryImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	m := f.GetFirstCell()
	if m == nil {
		return fmt.Errorf("No cells in the factory chain")
	}
	return m.HandleRequest(req, replyTo)
}

// loadCell starts an instance of a cell.
//
// If the cell implements the ITransportServer interface it is added to the list of available
// transport. See GetTransportServers() to obtain the collection of all loaded servers.
func (f *FactoryImpl) loadCell(cellType string) (m api.IHiveCell, isNew bool, err error) {
	f.mux.RLock()
	m, ok := f.singletonCells[cellType]
	f.mux.RUnlock()
	if m != nil {
		return m, false, nil
	}

	def, ok := f.cellDefinitions[cellType]
	if !ok {
		err := fmt.Errorf("loadCell: cell '%s' not found", cellType)
		return nil, false, err
	}
	// ignore empty slots
	if def.Constructor == nil {
		return nil, false, nil
	}
	slog.Info("loadCell loaded new cell instance", "cellType", cellType)
	cellInstance, err := def.Constructor(f, &def)

	if err != nil {
		return nil, false, err
	}
	// if nil is returned then nothing to do
	// this can be valid for initialization cells
	if cellInstance == nil {
		return cellInstance, false, nil
	}

	// store the singleton on successful start

	f.mux.Lock()
	f.singletonCells[cellType] = cellInstance
	tp, ok := cellInstance.(api.ITransportServer)
	if ok {
		f.transportCells = append(f.transportCells, tp)
	}
	f.mux.Unlock()

	// add to the loaded list
	f.mux.Lock()
	f.loadedCells = append(f.loadedCells, cellInstance)
	f.mux.Unlock()
	return cellInstance, true, nil
}

// RegisterCell registers a cell definition to the factory, making it available for creation.
// Used for registring recipe cells and support for 3rd party cells.
//
// If the given cellDef has a no factory function then only the config is added used.
func (f *FactoryImpl) RegisterCell(cellDef api.CellDefinition) {
	f.mux.Lock()
	defer f.mux.Unlock()
	// merge the registration if it exists
	// intended to preregister the cells and only use type definitions in the recipe
	existing, found := f.cellDefinitions[cellDef.Type]
	if found && cellDef.Constructor == nil {
		cellDef.Constructor = existing.Constructor
	}
	f.cellDefinitions[cellDef.Type] = cellDef
}

// Set the authenticator to use with the cell.
// Intended to be set by a service like authn that performs actual authentication.
// If nil is provided then disable authentication
func (f *FactoryImpl) SetAuthenticator(impl api.IAuthenticator) {
	f.authProxy.SetAuthenticator(impl)
}

// Stop all cells in reverse order
func (f *FactoryImpl) Stop() {
	n := len(f.loadedCells)
	slog.Info("StopAll: stopping all loaded cells", "count", n)
	for i := n - 1; i >= 0; i-- {
		m := f.loadedCells[i]
		m.Stop()
	}
	f.loadedCells = make([]api.IHiveCell, 0)
}

// Startcell loads and starts an instance of a cell by its type.
// If the cell is already started then it is returned as-is.
//
// This can return nil without error if the cell is a 'one-shot' cell whose
// factory function returns nil. Intended for initializing the factory environment.
//
// This returns an error if instantiate is false and the cell is not yet loaded.
func (f *FactoryImpl) StartCell(cellType string, instantiate bool) (api.IHiveCell, error) {
	f.mux.RLock()
	instance, ok := f.singletonCells[cellType]
	f.mux.RUnlock()

	// if the cell is already loaded, return it
	// if not loaded and instantiate is false then this is an error
	if instance != nil && ok {
		return instance, nil
	} else if !instantiate {
		return nil, fmt.Errorf("StartCell: Cell '%s' not yet loaded and instantiate is false", cellType)
	}

	instance, isNew, err := f.loadCell(cellType)
	_ = isNew
	if err != nil {
		return nil, err
	}
	return instance, err
}

// Wait for an OS signal or until the context is cancelled
func (f *FactoryImpl) WaitForSignal(ctx context.Context) {

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

// Start a new cell factory.
// Cells can be nil if they are registered separately or if StartRecipe is used.
//
//	env is the application enviroment created with api.NewAppEnvironment
//	cellDefs are the cell definitions available to GetCell(type)
func StartCellFactoryImpl(
	env *api.HiveEnvironment, cellDefs []api.CellDefinition) api.ICellFactory {

	cellDefMap := make(map[string]api.CellDefinition)
	for _, def := range cellDefs {
		cellDefMap[def.Type] = def
	}
	thingID := "factory"
	f := &FactoryImpl{
		HiveCellBase:    cells.NewHiveCellBase(thingID, env.RpcTimeout),
		authProxy:       NewAuthenticatorProxy(),
		env:             env,
		cellDefinitions: cellDefMap,
		singletonCells:  make(map[string]api.IHiveCell),
	}
	var _ api.ICellFactory = f // API check
	return f
}
