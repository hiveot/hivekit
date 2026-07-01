package factorypkg

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/factory/internal"
)

// Create a recipe instance for running modules in a chain formation.
//
// Use Start to instantiate and link the modules in the given order. This uses the factory
// to create the module instances.
// Providing a factory function in the chain is only needed if the factory doesn't contain
// it already.
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
