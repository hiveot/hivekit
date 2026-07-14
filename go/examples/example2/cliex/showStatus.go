package cliex

import (
	"fmt"
)

// Show the status of a Thing
//
// The thing must have published its TD using discovery.
//
//	thingID whose status to show
//	subscribe to property updates and events
func (app *Cliex) ShowStatus(thingID string, subscribe bool) {

	tdoc := app.FindTD(thingID)
	if tdoc == nil {
		fmt.Println("ShowStatus TD not found")
		return
	}
	fmt.Printf("Found the TD of Thing '%s'\n", thingID)

	// 2. import the TD into the directory client cache
	app.dirClient.Cache().ImportTD(tdoc)

	// 3: the router can query the thing using the discovered TD
	// the router has a credentials store for each known thingID
	// in order to be able to connect, that store has to be pre-configured with thingID
	values, err := app.co.ReadAllProperties(thingID)
	if err != nil {
		println("Error reading properties: " + err.Error())
		return
	}
	println("Properties:")
	for k, v := range values {
		fmt.Printf(" %s: %v\n", k, v)
	}
	notifs, err := app.co.ReadAllEvents(thingID)
	if err != nil {
		println("Error reading events: " + err.Error())
		return
	}
	println("Events:")
	for k, v := range notifs {
		fmt.Printf(" %s: Submitted at %v: %s\n", k, v.Timestamp, v.ToString(100))
	}
}
