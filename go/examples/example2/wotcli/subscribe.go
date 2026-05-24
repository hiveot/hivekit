package wotcli

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/examples/wotco"
	"github.com/hiveot/hivekit/go/utils"
)

// subscribe to events and observe properties
func Subscribe(co *wotco.WotConsumer, thingID string) {

	println("Subscribing to events and observing properties. Hit Ctrl-C to stop.")

	err := co.Subscribe(thingID, "")
	if err == nil {
		err = co.ObserveProperty(thingID, "")
	}
	if err != nil {
		println("Error reading properties: " + err.Error())
		return
	}
	co.SetAppNotificationHook(func(notif *msg.NotificationMessage) {
		fmt.Printf("Received notification: %s\n", notif.Name)
	})
	utils.WaitForSignal()
}
