package factory_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/factory/internal"
)

// Create a new cell factory.
// Cells can be nil if they are registered separately or if StartRecipe is used.
//
//	env is the application enviroment created with api.NewAppEnvironment
//	cellDefs are the cell definitions available to GetCell(type)
func NewCellFactory(
	env *api.HiveEnvironment,
	cellDefs []api.CellDefinition) api.ICellFactory {

	f := internal.NewCellFactoryImpl(env, cellDefs)
	return f
}
