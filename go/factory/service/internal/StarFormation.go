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
// If no member matches the request thingID then the request is forwarded to the
// formation sink.
//
// This is intended for grouping services where there is no need to pass requests
// through each service. Only the addressed service receives the request.
// Functionally a chain formation behaves the same but is less efficient.
//
// If a request is received for a thingID not in the formation, it is forwarded to the
// formation registered sink.
//
// The star recipe itself is registered as the notification sink of the cells in the
// star and will forward these notifications to its own registered notification sink.
//
// Star members will have forwarding disabled to avoid multiple notifications and requests.
type StarFormation struct {
	*cells.HiveCellBase
	// cells in the order to instantiate and link
	cellDefs []api.CellDefinition `yaml:"star"`

	// The factory to use
	f api.ICellFactory

	// cell members by their ThingID
	members map[string]api.IHiveCell
}

// Receives notifications from downstream and send it to all cells
func (r *StarFormation) HandleNotification(notif *msg.NotificationMessage) {
	for _, member := range r.members {
		member.HandleNotification(notif)
	}
}

// Requests sent to the star are passed on to the cell with the matching thingID.
// If no cells match it is forwarded to the registered sink.
func (r *StarFormation) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	cell, found := r.members[req.ThingID]
	if found {
		return cell.HandleRequest(req, replyTo)
	}
	return r.HiveCellBase.HandleRequest(req, replyTo)
}

// Update the member's sink for notifications from the formation.
func (r *StarFormation) SetNotificationSink(sink api.IHiveCell, thingIDs ...string) {
	for _, member := range r.members {
		member.SetNotificationSink(sink, thingIDs...)
	}
	r.HiveCellBase.SetNotificationSink(sink, thingIDs...)
}

// Set the sink for requests from the formation members
func (r *StarFormation) SetRequestSink(sink api.IHiveCell) {
	for _, member := range r.members {
		member.SetRequestSink(sink)
	}
	r.HiveCellBase.SetRequestSink(sink)
}

func (r *StarFormation) SetSlot(slotID string, modDef api.CellDefinition) error {
	for i, md := range r.cellDefs {
		if md.Type == slotID {
			r.cellDefs[i] = modDef
			return nil
		}
	}
	return fmt.Errorf("SetSlot: slot '%s' not found", slotID)
}

// NewStarFormation returns a ready-to-use formation with cells linked in a star.
//
// This returns the star formation cell.
func NewStarFormation(
	f api.ICellFactory, cellDefs []api.CellDefinition) (*StarFormation, error) {

	star := &StarFormation{
		HiveCellBase: cells.NewHiveCellBase(""),
		f:            f,
		cellDefs:     cellDefs,
		members:      make(map[string]api.IHiveCell),
	}
	star.SetTimeout(f.GetEnvironment().RpcTimeout)

	// add the cell definitions to the factory
	if star.cellDefs != nil {
		// register all cells
		for _, modDef := range star.cellDefs {
			star.f.RegisterCell(modDef)
		}
	}
	// create cells in the defined order and link their notifications
	for _, cellDef := range star.cellDefs {
		member, err := star.f.NewCell(cellDef.Type, true)
		// cell that cant be created are ignored. This is non-fatal
		if err != nil {
			slog.Warn("NewStarFormation: creating cell failed. Shutting down",
				"cellType", cellDef.Type, "err", err.Error())
		} else if member == nil {
			// don't track 'one-shot' cells that are used to initialize the factory.
			// These return nil without error.
		} else {
			// Members MUST not forward unhandled requests nor notifications,
			// otherwise the same message will be received multiple times.
			member.SetForwarding(false, false)
			star.members[member.GetThingID()] = member
			// // requests emitted by the members are forwarded to the formation sink.
			// member.SetRequestSink(star.GetRequestSink())
			// // notifications emitted by the cells are forwarded to the star notification sink.
			// member.SetNotificationSink(star.GetNotificationSink())
		}
	}

	var _ api.IHiveCell = star // interface check
	return star, nil
}
