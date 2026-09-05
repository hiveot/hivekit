package internal

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells"
)

// The StarFormation links its cells in a star formation.
//
// Incoming requests are forwarded to the cell that matches the request thingID.
// There is no need for linking individual request handlers.
//
// If a request is received for a thingID not in the star, it is forwarded to the
// star recipe registered sink.
//
// The star recipe itself is registered as the notification sink of the cells in the
// star and will forward these notifications to its own registered notification sink.
type StarFormation struct {
	*cells.HiveCellBase
	// cells in the order to instantiate and link
	star []api.CellDefinition `yaml:"star"`

	// The factory to use
	f api.ICellFactory

	// cell instances by their ThingID
	instances map[string]api.IHiveCell
}

// Receives notifications from downstream and send it to all cells
func (r *StarFormation) HandleNotification(notif *msg.NotificationMessage) {
	for _, member := range r.instances {
		member.HandleNotification(notif)
	}
}

// Requests sent to the star are passed on to the cell with the matching thingID.
// If no cells match it is forwarded to the registered sink.
func (r *StarFormation) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	ray, found := r.instances[req.ThingID]
	if found {
		return ray.HandleRequest(req, replyTo)
	}
	return r.HiveCellBase.HandleRequest(req, replyTo)
}

// Invoke Ready on all members of this formation
func (r *StarFormation) Ready() {
	for _, cell := range r.instances {
		cell.Ready()
	}
}

func (r *StarFormation) SetSlot(slotID string, modDef api.CellDefinition) error {
	for i, md := range r.star {
		if md.Type == slotID {
			r.star[i] = modDef
			return nil
		}
	}
	return fmt.Errorf("SetSlot: slot '%s' not found", slotID)
}

// StartStarFormation returns a formation with cells linked in a star.
//
// Call Ready when the application is ready to go. This calls Ready on all cells.
//
// This returns the star formation cell.
func StartStarFormation(
	f api.ICellFactory, members []api.CellDefinition) (*StarFormation, error) {

	r := &StarFormation{
		HiveCellBase: cells.NewHiveCellBase("", 0),
		f:            f,
		star:         members,
	}

	// add the cell definitions to the factory
	if r.star != nil {
		// register all cells
		for _, modDef := range r.star {
			r.f.RegisterCell(modDef)
		}
	}
	// start cells in the defined order and link their notifications
	for _, cellDef := range r.star {
		member, err := r.f.StartCell(cellDef.Type, true)
		// cell cant be started. This is fatal
		if err != nil {
			slog.Error("StartRecipe: starting cell failed. Shutting down",
				"cellType", cellDef.Type, "err", err.Error())
			r.Stop()
			return nil, err
		} else if member == nil {
			// don't track 'one-shot' cells that are used to initialize the factory.
			// These return nil without error.
		} else {
			r.instances[member.GetThingID()] = member
			// requests send by the members will be forwarded to the recipe, which
			// passes it to the member with the matching thingID. See HandleRequest.
			member.SetRequestSink(r)
			// all notifications from the rays will be forwarded to the star. See HandleNotification.
			member.SetNotificationSink(r)
		}
	}

	var _ api.IRecipe = r
	return r, nil
}
