package factory_service

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/factory/internal"
)

// Create a new module bus with an array of modules defined in the def config
func NewBusRecipeFactory(
	f api.IModuleFactory, def *api.ModuleDefinition) (api.IHiveModule, error) {

	members, ok := def.Config.([]api.ModuleDefinition)
	if !ok {
		return nil, fmt.Errorf("NewBusRecipeFactory: Config has no members")
	}
	m := internal.NewBusRecipe(members)
	return m, nil
}

// Create a recipe instance for running modules in a chain formation.
//
// Use Start to instantiate and link the modules in a chain.
// This uses the factory to create the module instances.
//
// f is the module factory that instantiates the modules
// chain is a collection of modules in order of instantiation.
//
// This returns the chain recipe module.
func NewChainRecipe(f api.IModuleFactory, chain []api.ModuleDefinition) api.IRecipe {

	m := internal.NewChainRecipe(f, chain)
	return m
}

// Create a recipe instance for running modules in a star formation.
// This returns the star recipe module.
func NewStarRecipe(
	f api.IModuleFactory, star []api.ModuleDefinition) api.IRecipe {

	m := internal.NewStarRecipe(f, star)
	return m
}
