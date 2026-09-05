package factory_service

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/factory/internal"
)

// StartBusFormation creates and starts cells in a bus formation.
//
// Members should not emit requests until Ready is invoked.
//
// Members will have their 'forward' capability disabled.
//
//   - A request sent to the formation is passed to each member until one accepts it.
//     Servers do not forward unhandled requests to their sink. They send them to clients or
//     return an 'undelivered' error.
//   - A request received by members is passed to the recipe's request sink.
//   - A notification sent to the recipe is passed to all members concurrently.
//     Servers do not forward notifications to their sink but to the remote connections instead.
//   - A notification received by members is passed to the recipe's notification sink.
func StartBusFormation(
	f api.ICellFactory, cellDefs []api.CellDefinition) (api.IRecipe, error) {

	bus, err := internal.StartBusFormation(f, cellDefs)
	return bus, err
}

// StartBusFormationFactory starts a new bus formation.
//
// Cells should not emit requests until Ready is invoked.
//
// * Both requests and notifications sent to the bus will be passed to all
// the members
// * Requests received from the bus will be forwarded to the bus request sink.
// * Notifications received from the bus will be forwarded to the bus notification sink.
//
// cellDef contains a list of CellDefinitions with the bus members.
func StartBusFormationFactory(
	f api.ICellFactory, cellDef *api.CellDefinition) (api.IHiveCell, error) {

	members, ok := cellDef.Config.([]api.CellDefinition)
	if !ok {
		return nil, fmt.Errorf("NewBusRecipeFactory: Config has no members")
	}
	bus, err := StartBusFormation(f, members)
	return bus, err
}

// StartChainFormation returns a collection of cells linked in a chain formation.
// Cells are started in the provided order.
//
// Cells should not emit requests until Ready is invoked.
//
//	f is the cell factory that instantiates the cells
//	cells is a collection of cells in order of instantiation.
//	linkTo is the optional request sink of this chain, and source of notifications.
//		A call to Ready and Stop will also be passed to the linkTo cell.
//
// This returns the chain formation as a recipe instance.
func StartChainFormation(
	f api.ICellFactory, cellDefs []api.CellDefinition, linkTo api.IHiveCell) (api.IRecipe, error) {

	chain, err := internal.StartChainFormation(f, cellDefs, linkTo)
	return chain, err
}
