package cliex

import (
	"fmt"

	"github.com/hiveot/hivekit/go/utils"
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
		fmt.Printf("ShowStatus TD '%s' not found\n", thingID)
		return
	}
	fmt.Printf("Found the TD of Thing '%s'\n", thingID)

	// 2. import the TD into the directory client cache
	app.dirClient.Cache().ImportTD(tdoc)

	fmt.Printf("Title '%s'\n", tdoc.Title)
	fmt.Printf("Modified '%s'\n", tdoc.Modified)

	// 3: the router can query the thing using the discovered TD
	// the router has a credentials store for each known thingID
	// in order to be able to connect, that store has to be pre-configured with thingID
	propValues, err := app.co.ReadAllProperties(thingID)
	if err != nil {
		println("Error reading properties: " + err.Error())
		return
	}
	fmt.Printf("Read-only Properties:\n")
	sortedKeys := utils.OrderedMapKeys(tdoc.Properties)
	for _, k := range sortedKeys {
		v, _ := propValues[k]
		propAff := tdoc.GetProperty(k)
		if propAff.ReadOnly {
			fmt.Printf(" %-30s - %-40s: %v\n", k, propAff.Title, v)
		}
	}

	fmt.Printf("\nWritable Properties:\n")
	for _, k := range sortedKeys {
		v, _ := propValues[k]
		propAff := tdoc.GetProperty(k)
		if !propAff.ReadOnly {
			fmt.Printf(" %-30s - %-40s: %v\n", k, propAff.Title, v)
		}
	}

	fmt.Printf("\nEvents (%d):\n", len(tdoc.Events))
	notifs, err := app.co.ReadAllEvents(thingID)
	if err != nil {
		println("Error reading events: " + err.Error())
	}
	sortedKeys = utils.OrderedMapKeys(tdoc.Events)
	for _, k := range sortedKeys {
		v, _ := notifs[k]
		if v == nil {
			fmt.Printf(" %s: No value\n", k)
		} else {
			fmt.Printf(" %s: Submitted at %v: %s\n", k, v.Timestamp, v.ToString(100))
		}
	}
}
