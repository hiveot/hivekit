package wotcli

import (
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/examples/wotco"
	"github.com/hiveot/hivekit/go/utils"
)

// subscribe to events and observe properties
func Subscribe(co *wotco.WotConsumer, thingID string) {
	fmt.Printf("Running discovery...   ")
	err := co.Discover(nil)
	if err != nil {
		fmt.Printf("Discovery failed: %v\n", err)
		return
	}
	println("Subscribing to events and observing properties. Hit Ctrl-C to stop.")

	err = co.Subscribe(thingID, "")
	if err == nil {
		err = co.ObserveProperty(thingID, "")
	}
	if err != nil {
		println("Error reading properties: " + err.Error())
		return
	}
	co.SetAppNotificationHook(func(notif *msg.NotificationMessage) {
		ts := time.Now().Local().Format(time.TimeOnly)
		fmt.Printf("%s: Received notification '%s %s': %s\n", ts, notif.AffordanceType, notif.Name, notif.ToString(20))
	})

	// FIXME: Consumer to detect a disconnect and resubscribe

	utils.WaitForSignal()
}
