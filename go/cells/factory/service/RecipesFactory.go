package factory_service

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/factory/internal"
)

// Create a new cell bus with an array of cells defined in the def config
func NewBusRecipeFactory(
	f api.ICellFactory, cellDef *api.CellDefinition) (api.IHiveCell, error) {

	members, ok := cellDef.Config.([]api.CellDefinition)
	if !ok {
		return nil, fmt.Errorf("NewBusRecipeFactory: Config has no members")
	}
	m := internal.NewBusRecipe(members)
	return m, nil
}

// Create a recipe instance for running cells in a chain formation.
//
// Use Start to instantiate and link the cells in a chain.
// This uses the factory to create the cell instances.
//
//	f is the cell factory that instantiates the cells
//	chain is a collection of cells in order of instantiation.
//
// This returns the chain recipe instance.
func NewChainRecipe(f api.ICellFactory, chain []api.CellDefinition) api.IRecipe {

	m := internal.NewChainRecipe(f, chain)
	return m
}

// Create a recipe instance for running cells in a star formation.
// This returns the star recipe instance.
func NewStarRecipe(
	f api.ICellFactory, star []api.CellDefinition) api.IRecipe {

	m := internal.NewStarRecipe(f, star)
	return m
}
