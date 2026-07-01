package factorypkg

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/factory/internal"
)

// Create a new module factory.
// Modules can be nil if they are registered separately or if StartRecipe is used.
//
//	env is the application enviroment created with api.NewAppEnvironment
//	moduleDefs are the module definitions available to GetModule(type)
func NewModuleFactory(
	env *api.AppEnvironment,
	moduleDefs []api.ModuleDefinition) api.IModuleFactory {

	f := internal.NewModuleFactoryImpl(env, moduleDefs)
	return f
}
