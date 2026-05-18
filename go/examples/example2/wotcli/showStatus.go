package wotcli

import (
	"fmt"

	"github.com/hiveot/hivekit/go/examples/wotco"
)

// Show the status of a Thing
//
//	thingID whose status to show
//	subscribe to property updates and events
func ShowStatus(co *wotco.WotConsumer, thingID string, subscribe bool) {

	// first discover the TDs
	err := co.Discover(nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	} else if co.GetTD(thingID) == nil {
		fmt.Printf("Thing with ID '%s' not discovered\n", thingID)
		return
	}

	values, err := co.ReadAllProperties(thingID)
	if err != nil {
		println("Error reading properties: " + err.Error())
		return
	}
	println("Properties:")
	for k, v := range values {
		fmt.Printf(" %s: %v\n", k, v)
	}
	notifs, err := co.ReadAllEvents(thingID)
	if err != nil {
		println("Error reading events: " + err.Error())
		return
	}
	println("Events:")
	for k, v := range notifs {
		fmt.Printf(" %s: Submitted at %v: %s\n", k, v.Timestamp, v.ToString(100))
	}
}
