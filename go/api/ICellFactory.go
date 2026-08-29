package api

import (
	"context"

	"github.com/hiveot/hivekit/go/api/td"
)

// the constructor function to create an instance of the cell using the given environment
// The recommended cellID is auto-generated. The cell can decide to override if needed.
// type CellFactoryFn func(f ICellFactory) api.IHiveCell

// CellDefinition defines the constructor for a cell, used for registration in the cell factory
// This can also be used to add custom cells.
type CellDefinition struct {

	// Set Multiton to true to allow multiple instances of the cell.
	// Multiton instances of the cell require different instance IDs.
	// Multiton bool

	// Type of the cell, used for registration and lookup.
	// Note that the cell type is identical for all instances of a cell and is used in the @type
	// field of the cell TM, if used. The cellID is the instance ID of the cell and
	// must be unique. Singleton cells use the same ID for cell type and cellID.
	Type string

	// The constructor function to create an instance of the cell.
	// The configuration can be used to pass arguments and other configuration to the cell.
	//
	// 	f is the cell factory with the app environment and ability to retrieve other cells
	// 	modDef is the cell definition passed to the constructor.
	//
	// This returns an error if the cell cannot be created.
	// This returns nil with no error for cells that are used for initialization.
	Constructor func(f ICellFactory, modDef *CellDefinition) (IHiveCell, error)

	// Optional configuration passed to the creation of the cell
	Config any
}

// ICellFactory is the interface for the cell factory, used to create and manage
// cells by their type.
//
// The cell factory can be used stand-alone or together with the ChainRecipe or StarRecipe.
type ICellFactory interface {

	// Add security and forms to the TD for all running transport protocols
	// Intended for devices to add forms before exporting a TD.
	// This passes the request to all server instances that have been created using
	// this factory.
	AddTDSecForms(tdoc *td.TD, includeAffordances bool)

	// Provide the means to authenticate incoming connections.
	// Intended for transport server cells.
	// This returns a proxy stub that can be updated with SetAuthenticator.
	// If no authenticator is set the this proxy fails all authentication attempts.
	//
	// SetAuthenticator is called by the authn cell when it is created.
	GetAuthenticator() IAuthenticator

	// Get the connection URL of the loaded transport servers or nil if none.
	// Primarily intended for testing. It is recommended to use a discovery server/client cell
	// in the factory server/client chains to facilitate discovery of server by the client.
	GetConnectURLs() []string

	// GetEnvironment returns the application environment used by the factory for
	// confuring cells.
	// Note that the environment can be updated by the cells to allow factory cells
	// to update the TDD, location of gateway and other discoverable information.
	GetEnvironment() *HiveEnvironment

	// GetCell returns the loaded cell of the given cell type.
	//
	// If a cell hasn't been loaded/started yet then this returns nil.
	GetCell(cellType string) IHiveCell

	// Return the http server cell instance.
	//
	// Used for cells that need to serve http endpoints, e.g. http basic authn, directory, etc.
	//
	// Set the instantiate flag to indicate that the http server cell of type TLSServerCellType
	// should be loaded if it hasn't been loaded yet. If no such cell is registered in the factory
	// cell definitions then this returns nil and a warning is logged.
	//
	//  instantiate set to true to auto load the http server cell
	//
	// This returns nil if no httpserver cell is registered.
	GetHttpServer(instantiate bool) IHttpServer

	// Obtain the directory TD.
	// Intended for bootstrapping the directory client.
	// GetTDD() *td.TD

	// Return the list of available transport servers
	GetTransportServers() []ITransportServer

	// RegisterCell adds a cell to the factory, making it available for instantiation
	// and for running recipes.
	//
	// If a cell is already registered it is replaced. If the given definition
	// doesn't contain a factory constructor but the existing registration does then
	// only the config from the definition is used and merged with the existing registration.
	//
	// Intended to allow pre-registering cells and only include a ordered list of
	// cells in the chain to instantiate and link.
	//
	// cellDef defines the cell attributes and constructor function
	RegisterCell(cellDef CellDefinition)

	// SetAuthenticator sets the authenticator returned by GetAuthenticator.
	// Note that GetAuthenticator returns a proxy to the actual authenticator.
	// Intended for use by the cell that offers authentication capabilities,
	// such as the authn cell.
	//
	// By default the authenticator proxy blocks all authentication.
	// Setting a nil authenticator disables authentication.
	SetAuthenticator(a IAuthenticator)

	// StartCell creates and starts an instance of a cell by its type.
	//
	// If the cell is already started, the existing cell instance is returned.
	//
	// If the cell factory function is nil then this is an empty slot which
	// will be ignored.
	//
	// This does not link the cell to other cells. See also RunRecipe for creating a chain.
	//
	//  cellType identifies the type of the cell to get.
	//	instantiate set to true to create an instance if one isnt loaded
	//
	// This returns an error if no cell with the given type is found, or when
	// starting the cell fails.
	// This returns nil with no error if the cell factory is a 'one-shot'
	// initialization function where its factory handler returns nil.
	StartCell(cellType string, instantiate bool) (IHiveCell, error)

	// Stop all loaded cells in reverse order of loading.
	// Intended for graceful shutdown.
	Stop()

	// WaitForSignal waits until an OS SigTerm signal is received or context is cancelled.
	// Call StopAll() afters this returns for proper cleanup.
	WaitForSignal(ctx context.Context)
}

// Helper to get a cell from the factory with the given interface
func GetFactoryCell[T interface{}](f ICellFactory, cellType string) T {
	m := f.GetCell(cellType)
	t, _ := m.(T)
	return t
}
