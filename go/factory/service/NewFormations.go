package factory_service

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/factory/service/internal"
)

// NewBusFormation creates and starts cells in a bus formation.
//
// Members should not emit requests autonomously until after Start is invoked.
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
func NewBusFormation(
	f api.ICellFactory, cellDefs []api.CellDefinition) (api.IHiveCell, error) {

	bus, err := internal.NewBusFormation(f, cellDefs)
	return bus, err
}

// NewBusFormationFactory starts a new bus formation.
//
// Cells should not emit requests autonomously until after Start is invoked.
//
// * Both requests and notifications sent to the bus will be passed to all
// the members
// * Requests received from the bus will be forwarded to the bus request sink.
// * Notifications received from the bus will be forwarded to the bus notification sink.
//
// cellDef contains a list of CellDefinitions with the bus members.
func NewBusFormationFactory(
	f api.ICellFactory, cellDef *api.CellDefinition) (api.IHiveCell, error) {

	members, ok := cellDef.Config.([]api.CellDefinition)
	if !ok {
		return nil, fmt.Errorf("NewBusRecipeFactory: Config has no members")
	}
	bus, err := internal.NewBusFormation(f, members)
	return bus, err
}

// NewChainFormation returns a collection of cells linked in a chain formation.
// Cells are created in the provided order.
//
// Members should not emit requests autonomously until after Start is invoked.
//
//	f is the cell factory that instantiates the cells
//	cells is a collection of cells in order of instantiation.
//	linkTo is the optional request sink of this chain, and source of notifications.
//		A call to Start and Stop will also be passed to the linkTo cell.
//
// This returns the chain formation as a recipe instance.
func NewChainFormation(
	f api.ICellFactory, cellDefs []api.CellDefinition, linkTo api.IHiveCell) (api.IHiveCell, error) {

	chain, err := internal.NewChainFormation(f, cellDefs, linkTo)
	// warning, chain is a pointer, returning it as an interface when the pointer is nil
	// no longer evaluates to nil after returning as IHiveCell. The reason is that the interface is
	// still intact even though the intstance is nil.
	if err != nil {
		return nil, err
	}
	return chain, nil
}

func NewChainFormationFactory(
	f api.ICellFactory, cellDef *api.CellDefinition) (api.IHiveCell, error) {

	members, ok := cellDef.Config.([]api.CellDefinition)
	if !ok {
		return nil, fmt.Errorf("NewChainFormationFactory: Config has no members")
	}
	bus, err := internal.NewChainFormation(f, members, nil)
	if err != nil {
		return nil, err
	}
	return bus, err
}

// NewStarFormation returns a ready-to-use formation for running cells in a star.
//
// Requests are passed to the member with the matching ThingID.
// Notifications passed to the formation are passed to each cell. Forwarding on the
// cell is disabled.
//
// Call Start() on the factory to activate autonomous processes and publications.
//
// Use SetNotificationSink and SetRequestSink before calling Start on the factory.
//
// If a cell fail to be created then this continues without the failed cell.
func NewStarFormation(
	f api.ICellFactory, members []api.CellDefinition) (api.IHiveCell, error) {

	star, err := internal.NewStarFormation(f, members)
	if err != nil {
		return nil, err
	}
	return star, err
}

// NewStarFormationFactory starts a new star formation.
//
// This formation is a cell that directs requests to the matching thingID.
//
// * Notifications of cells are passed to the formation notification handler.
// * Notifications received by the formation are passed to all cells.
// * Requests from cells are passed to the formation request sink.
// * Unhandled requests are forwarded to the sink unless forwarding is disabled.
// * Cells have their notification forwarding disabled to avoid duplicates.
//
// cellDef contains a list of CellDefinitions with the star members.
func NewStarFormationFactory(
	f api.ICellFactory, cellDef *api.CellDefinition) (api.IHiveCell, error) {

	members, ok := cellDef.Config.([]api.CellDefinition)
	if !ok {
		return nil, fmt.Errorf("NewStarFormationFactory: Config has no members")
	}
	bus, err := internal.NewStarFormation(f, members)
	if err != nil {
		return nil, err
	}
	return bus, err
}
