package factory_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/factory/internal"
)

// Start a new cell factory.
// Cells can be nil if they are registered separately or if StartRecipe is used.
//
//	env is the application enviroment created with api.NewAppEnvironment
//	cellDefs are the cell definitions available to GetCell(type)
func StartCellFactory(
	env *api.HiveEnvironment,
	cellDefs []api.CellDefinition) api.ICellFactory {

	f := internal.StartCellFactoryImpl(env, cellDefs)
	return f
}
