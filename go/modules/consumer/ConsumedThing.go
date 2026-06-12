package consumer

import (
	"fmt"

	"github.com/hiveot/hivekit/go/modules"
)

// A consumedThing is a local representation of a remote Thing
// It should be linked to a client that provides the actual connection.
//
// Usage
type ConsumedThing struct {
	modules.HiveModuleBase
}

// Start the module. This subscribes to
func (ct *ConsumedThing) Start() error {
	return fmt.Errorf("not yet implemented")
}

func NewConsumedThing(thingID string, sink modules.IHiveModule) *ConsumedThing {
	ct := &ConsumedThing{
		HiveModuleBase: *modules.NewHiveModuleBase(thingID, 0),
	}
	if sink != nil {
		ct.SetRequestSink(sink)
		sink.SetNotificationSink(ct)
	}
	return ct
}
