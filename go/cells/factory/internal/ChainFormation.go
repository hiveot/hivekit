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
// NOTE: The chain recipe MUST be linked before start. On Start the last cell
// of the chain is set to the linked request handler.
//
// On start, all cells are loaded, started and linked in sequence.
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
	modList []api.IHiveCell
}

// Recipe receives notifications from the application.
// Send it up the recipe content chain, starting at the last cell.
func (r *ChainFormation) HandleNotification(notif *msg.NotificationMessage) {
	if len(r.modList) == 0 {
		return
	}
	tail := r.modList[len(r.modList)-1]
	tail.HandleNotification(notif)
}

// Requests sent to the chain are passed on to the first cell in the chain.
// If no cells are registered then this is an error.
func (r *ChainFormation) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	if len(r.modList) == 0 {
		return fmt.Errorf("HandleRequest: recipe has no cells registered")
	}
	head := r.modList[0]
	return head.HandleRequest(req, replyTo)
}

// Set the sink for notifications from the chain
// This sets the sink to the first cell in the chain. Call this after start.
func (r *ChainFormation) SetNotificationSink(sink api.IHiveCell, thingIDs ...string) {
	if len(r.modList) == 0 {
		slog.Error("SetNotificationSink called but the chain is not started")
		return
	}
	head := r.modList[0]
	head.SetNotificationSink(sink, thingIDs...)
}

// Set the sink for requests from the chain
// This sets the sink to the last cell in the chain. Call this after start.
func (r *ChainFormation) SetRequestSink(sink api.IHiveCell) {
	if len(r.modList) == 0 {
		slog.Error("SetRequestSink called but the chain is not started")
		return
	}
	tail := r.modList[len(r.modList)-1]
	tail.SetRequestSink(sink)
}

// SetSlot sets the given cell definition in the chain at the position of the slot.
// Use this before starting the chain.
// Intended to create chain templates where the application cell needs to be placed
// before some other cells.
func (r *ChainFormation) SetSlot(slotID string, modDef api.CellDefinition) error {
	for i, md := range r.chain {
		if md.Type == slotID {
			r.chain[i] = modDef
			return nil
		}
	}
	return fmt.Errorf("SetSlot: slot '%s' not found", slotID)
}

// Start the recipe.
// This starts the cells in sequence.
//
// NOTE: The chain recipe must be started before linking to it, as setting the recipe request
// sink sets it on the last cell in the chain. and setting the notification sink sets it
// on the first cell of the chain:
//
// * linking a request handler sets its as the sink of the last cell
// * linking a notification handler sets it as the sink of the first cell
// * sending a request to the chain passes it to the first cell of the chain
// * sending a notification to the chain passes it to the last cell, which makes it
//
//	way to the first cell and up to the linked notification handler.
func (r *ChainFormation) Start() error {

	// register all cells with the factory
	for _, cellDef := range r.chain {
		r.f.RegisterCell(cellDef)
	}

	// start and link cells in the defined order
	r.modList = make([]api.IHiveCell, 0, len(r.chain))
	var prevCell api.IHiveCell
	for _, cellDef := range r.chain {
		member, err := r.f.StartCell(cellDef.Type, true)
		if err != nil {
			slog.Error("StartRecipe: starting cell failed. Shutting down",
				"cellType", cellDef.Type, "err", err.Error())
			r.Stop()
			return err
		} else if member == nil {
			// don't track 'one-shot' cells that are used to initialize the factory.
			// These return nil without error.
		} else {
			r.modList = append(r.modList, member)
			// Link the cell to the previous cell in the list
			if prevCell != nil {
				prevCell.SetRequestSink(member)
				member.SetNotificationSink(prevCell)
			}
		}
		prevCell = member
	}

	return nil
}

// Create a recipe instance for running cells in a chain formation.
//
// Use Start to instantiate and link the cells in the given order. This uses the factory
// to create the cell instances.
//
//	f is the cell factory that instantiates the cells
//	chain is a collection of cells in order of instantiation.
//
// This returns the chain recipe.
func NewChainFormation(f api.ICellFactory, chain []api.CellDefinition) *ChainFormation {

	r := &ChainFormation{
		HiveCellBase: cells.NewHiveCellBase("ChainRecipe", 0),
		f:            f,
		chain:        chain,
	}
	var _ api.IRecipe = r
	return r
}
