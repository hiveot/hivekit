package cliapp

import (
	"fmt"
	"time"
)

// Show the status of a Thing
//
// The thing must have published its TD using discovery.
//
//	thingID whose status to show
//	subscribe to property updates and events
func (app *CliApp) ShowStatus(thingID string, subscribe bool) {
	var maxWaitTime = time.Second * 3

	// 1. discover the things
	recs, err := app.discoClient.DiscoverThings("", maxWaitTime, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// 2. import the TD into the directory client cache
	for _, rec := range recs {
		tdURL := rec.AsURL()
		tdoc, _, err := app.discoClient.LoadTD(tdURL)
		_ = tdoc
		if err == nil {
			app.dirClient.Cache().ImportTD(tdoc)
		}
	}

	// 3: the router can query the thing using the discovered TD
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
