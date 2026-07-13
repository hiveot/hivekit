package cliex

import (
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/utils"
)

// subscribe to events and observe properties
func (app *Cliex) Subscribe(thingID string) {

	println("Subscribing to events and observing properties. Hit Ctrl-C to stop.")

	err := app.co.Subscribe(thingID, "")
	if err == nil {
		err = app.co.ObserveProperty(thingID, "")
	}
	if err != nil {
		println("Error reading properties: " + err.Error())
		return
	}
	app.co.SetNotificationHook(func(notif *msg.NotificationMessage) {
		ts := time.Now().Local().Format(time.TimeOnly)
		fmt.Printf("%s: Received notification '%s %s': %s\n", ts, notif.AffordanceType, notif.Name, notif.ToString(20))
	})

	// FIXME: Consumer to detect a disconnect and resubscribe

	utils.WaitForSignal()
}
