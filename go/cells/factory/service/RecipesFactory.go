package factory_service

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/factory/internal"
)

// NewBusFormation returns a collection of cells in a bus formation
//
// Characteristics:
//   - A request sent to the recipe is passed to each member until one accepts it.
//   - Servers do not forward unhandled requests to their sink. They send them to clients or
//     return an 'undelivered' error.
//   - A request received by members is passed to the recipe's request sink.
//   - A notification sent to the recipe is passed to all members concurrently.
//   - servers do not forward notifications to their sink but to the remote connections instead.
//   - A notification received by members is passed to the recipe's notification sink.
func NewBusFormation(
	f api.ICellFactory, cells []api.CellDefinition) (api.IHiveCell, error) {

	m := internal.NewBusFormation(f, cells)
	return m, nil
}

// NewBusFactory is the factory handler for creating cells in bus formation.
func NewBusFactory(
	f api.ICellFactory, cellDef *api.CellDefinition) (api.IHiveCell, error) {

	members, ok := cellDef.Config.([]api.CellDefinition)
	if !ok {
		return nil, fmt.Errorf("NewBusRecipeFactory: Config has no members")
	}
	m := internal.NewBusFormation(f, members)
	return m, nil
}

// NewChainFormation returns a collection of cells linked in a chain formation.
//
// Use Start to instantiate and link the cells in a chain.
// This uses the factory to create the cell instances.
//
//	f is the cell factory that instantiates the cells
//	cells is a collection of cells in order of instantiation.
//
// This returns the chain formation as a recipe instance.
func NewChainFormation(f api.ICellFactory, cells []api.CellDefinition) api.IRecipe {

	m := internal.NewChainFormation(f, cells)
	return m
}

// NewStarFormation returns a collection of cells in a star formation.
//
// This returns the 'cell star' formation as a recipe instance
func NewStarFormation(
	f api.ICellFactory, star []api.CellDefinition) api.IRecipe {

	m := internal.NewStarFormation(f, star)
	return m
}
