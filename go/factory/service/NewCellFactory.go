package factory_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/factory/service/internal"
)

// NewCellFactory returns a new cell factory.
// Cells can be nil if they are registered separately or if StartRecipe is used.
//
// All cells created will have their timeout set to that of env.RpcTimeout
//
//	env is the application enviroment created with api.NewAppEnvironment
//	cellDefs are the cell definitions available to GetCell(type)
func NewCellFactory(
	env *api.AppEnvironment,
	cellDefs []api.CellDefinition) api.ICellFactory {

	f := internal.NewCellFactoryImpl(env, cellDefs)
	return f
}
