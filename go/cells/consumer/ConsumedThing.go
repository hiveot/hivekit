package consumer

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells"
)

// A consumedThing is a local representation of a remote Thing
//
// This is work in progress. The intent is to subscribe to Thing updates and reflect its values.
// It should be linked to a client that provides the actual connection.
//
// Usage
type ConsumedThing struct {
	cells.HiveCellBase
}

// TODO: Start the consumed thing, subscribe, etc,etc
func (ct *ConsumedThing) Start() error {
	return fmt.Errorf("not yet implemented")
}

func NewConsumedThing(thingID string, sink api.IHiveCell) *ConsumedThing {
	ct := &ConsumedThing{
		HiveCellBase: *cells.NewHiveCellBase(thingID, 0),
	}
	if sink != nil {
		ct.SetRequestSink(sink)
		sink.SetNotificationSink(ct)
	}
	return ct
}
