package factorypkg

import (
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/factory/internal"
)

// Create a new module factory.
// Modules can be nil if they are registered separately or if StartRecipe is used.
//
//	env is the application enviroment created with factory.NewAppEnvironment
//	moduleDefs are the module definitions available to GetModule(type)
func NewModuleFactory(
	env *factory.AppEnvironment,
	moduleDefs []factory.ModuleDefinition) factory.IModuleFactory {

	f := internal.NewModuleFactoryImpl(env, moduleDefs)
	return f
}
