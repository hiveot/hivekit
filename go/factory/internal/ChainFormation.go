package internal

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells"
)

// The ChainFormation links its cells in a chain formation.
//
// Incoming requests are forwarded to the first cell in the chain.
// The last cell in the chain will have its request sink set to the sink of the
// chain itself, effectively forwarding requests to the request sink of the chain.
//
// The ChainFormation itself is registered as the notification sink of the first cell
// in the chain and will forward these notifications to its registered notification sink.
type ChainFormation struct {
	*cells.HiveCellBase
	// Chain of cells in the order to instantiate and link
	chain []api.CellDefinition `yaml:"chain"`

	// The factory to use
	f api.ICellFactory

	// loaded cells in order of the chain
	instances []api.IHiveCell

	// linkto provided on start.
	// this will also receive the Ready call.
	linkTo api.IHiveCell
}

// Recipe receives notifications from the application.
// Send it up the recipe content chain, starting at the last cell.
func (r *ChainFormation) HandleNotification(notif *msg.NotificationMessage) {
	if len(r.instances) == 0 {
		return
	}
	tail := r.instances[len(r.instances)-1]
	tail.HandleNotification(notif)
}

// Requests sent to the chain are passed on to the first cell in the chain.
// If no cells are registered then this is an error.
func (r *ChainFormation) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	if len(r.instances) == 0 {
		return fmt.Errorf("HandleRequest: recipe has no cells registered")
	}
	head := r.instances[0]
	return head.HandleRequest(req, replyTo)
}

// Invoke Ready on all members of this formation including the optional linkTo
func (r *ChainFormation) Ready() {
	for _, cell := range r.instances {
		cell.Ready()
	}
	if r.linkTo != nil {
		r.linkTo.Ready()
	}
}

// Set the sink for notifications from the chain
// This sets the sink to the first cell in the chain. Call this after start.
func (r *ChainFormation) SetNotificationSink(sink api.IHiveCell, thingIDs ...string) {
	if len(r.instances) == 0 {
		slog.Error("SetNotificationSink called but the chain is not started")
		return
	}
	head := r.instances[0]
	head.SetNotificationSink(sink, thingIDs...)
}

// Set the sink for requests from the chain.
// This sets the sink to the last cell in the chain.
// Only needed if no LinkTo is provided in ChainFormation
func (r *ChainFormation) SetRequestSink(sink api.IHiveCell) {
	if len(r.instances) == 0 {
		slog.Error("SetRequestSink called but the chain has no members")
		return
	}
	tail := r.instances[len(r.instances)-1]
	tail.SetRequestSink(sink)
}

// SetSlot sets the given cell definition in the chain at the position of the slot.
// Intended to create chain templates where the application cell needs to be placed
// before some other cells.
//
//	slotID is the slot to replace
//	cellDef is the cell to fill the slot with
func (r *ChainFormation) SetSlot(slotID string, cellDef api.CellDefinition) error {
	// locate the slit
	for i, md := range r.chain {
		if md.Type == slotID {
			r.chain[i] = cellDef
			// link

			return nil
		}

	}
	return fmt.Errorf("SetSlot: slot '%s' not found", slotID)
}

// Start a chain formation recipe for running cells linked in a chain.
//
// - HandleRequest will send the request to the first cell of the chain.
// - HandleNotification passes it to the last cell, which makes its way up the chain.
// - SetRequestSink is not needed if linkTo is provided
// - SetNotificationSink sets the destination of notification that are passed up the chain.
//
// Cells in the chain should not emit requests or notifications until the chain is
// linked as intended. If all cells are reactive, eg respond to external requests and
// notifications, this should not be an issue.
//
// Cells that publish a TD for updating a directory must wait with this until the
// environment is ready.
//
//	f is the cell factory that instantiates the cells
//	chain is a collection of cells in order of instantiation.
//	linkTo is optional cell that handles requests and passes notifications to the chain.
//		This can be used instead of SetRequestSink.
//
// This returns the chain recipe.
func StartChainFormation(
	f api.ICellFactory, chain []api.CellDefinition, linkTo api.IHiveCell) (*ChainFormation, error) {

	r := &ChainFormation{
		HiveCellBase: cells.NewHiveCellBase("ChainFormation", 0),
		f:            f,
		chain:        chain,
		linkTo:       linkTo,
	}

	// register all cells with the factory
	for _, cellDef := range r.chain {
		r.f.RegisterCell(cellDef)
	}

	// start and link cells in the defined order
	r.instances = make([]api.IHiveCell, 0, len(r.chain))
	var prevCell api.IHiveCell

	// Instantiate cells and link in the specified order.
	for _, cellDef := range r.chain {
		member, err := r.f.StartCell(cellDef.Type, true)
		if err != nil {
			slog.Error("Start: starting cell failed. Shutting down",
				"cellType", cellDef.Type, "err", err.Error())
			r.Stop()
			return nil, err
		} else if member == nil {
			// don't track 'one-shot' cells that are used to initialize the factory.
			// These return nil without error.
		} else {
			// success
			r.instances = append(r.instances, member)
			if prevCell != nil {
				// prevCell.SetRequestSink(member)
				member.SetNotificationSink(prevCell)
				prevCell.SetRequestSink(member)
			}
		}
		prevCell = member
	}

	// The recipe tail links to the linkTo cell.
	if linkTo != nil && len(r.instances) > 1 {
		tail := r.instances[len(r.instances)-1]
		linkTo.SetNotificationSink(tail)
		tail.SetRequestSink(linkTo)
	}

	var _ api.IRecipe = r
	return r, nil
}
