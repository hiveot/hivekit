package consumer

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/thing"
)

// A consumedThing is a local representation of a remote Thing
//
// This is work in progress. The intent is to subscribe to Thing updates and reflect its values.
// It should be linked to a client that provides the actual connection.
//
// Usage
type ConsumedThing struct {
	cells.HiveCellBase

	tdoc  *td.TD
	state thing.ExposedThingState
	// properties map[string]*msg.NotificationMessage
	// events     map[string]*msg.NotificationMessage

	// co IConsumer
}

// HandleNotification captures the values
func (ct *ConsumedThing) HandleNotification(notif *msg.NotificationMessage) {
	switch notif.AffordanceType {
	case msg.AffordanceTypeAction:
		// ct.actions[notif.Name] = notif
	case msg.AffordanceTypeEvent:
		ct.state.SetEvent(notif.Name, notif)
	case msg.AffordanceTypeProperty:
		ct.state.SetProperty(notif.Name, notif)
	}
}

// Return a new consumed thing.
// This subscribes to notification for the Things.
func NewConsumedThing(tdoc *td.TD, co IConsumer) (*ConsumedThing, error) {
	ct := &ConsumedThing{
		HiveCellBase: *cells.NewHiveCellBase(tdoc.ID),
	}

	co.Subscribe(tdoc.ID, "")
	co.ObserveProperty(tdoc.ID, "")

	if co != nil {
		ct.SetRequestSink(co)
		co.SetNotificationSink(ct, tdoc.ID)
	}
	// TODO: Start the consumed thing, subscribe, etc,etc

	// return ct
	return ct, fmt.Errorf("not yet implemented")
}
