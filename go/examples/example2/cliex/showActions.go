package cliex

import (
	"fmt"
)

// Show the actions of a Thing
//
// If a name is provided then invoke the action. Currently this doesn't accept input data.
//
// The thing must have published its TD using discovery.
//
//	thingID whose actions to show
//	actionName optional action name to invoke
func (app *Cliex) ShowActions(thingID string, actionName string) {

	tdoc := app.FindTD(thingID)
	if tdoc == nil {
		fmt.Printf("ShowAction TD for thing '%s' not found\n", thingID)
		return
	}
	fmt.Printf("Found the TD of Thing '%s'\n", thingID)

	// 2. import the TD into the directory client cache
	app.dirClient.Cache().ImportTD(tdoc)

	// 3. check for listing actions
	if actionName == "" {
		// show the action
		println("Actions:")
		for k, aff := range tdoc.Actions {
			fmt.Printf("  %s: %v\n", k, aff.Title)
		}
		return
	}
	aff := tdoc.Actions[actionName]
	if aff == nil {
		fmt.Printf("Action '%s' does not exist on thing '%s'", actionName, thingID)
		return
	}
	// check for input
	if aff.Input != nil {
		fmt.Printf("Sorry, capturing input for action '%s' is not yet supported.\n", actionName)
		return
	}
	// invoke the action
	fmt.Printf("Invoking action '%s' on '%s', thingID '%s'\n", actionName, tdoc.Title, thingID)
	var input any
	var output any

	//
	err := app.co.InvokeAction(thingID, actionName, input, &output)
	if err != nil {
		fmt.Printf("InvokeAction '%s' returned error: %s\n", actionName, err.Error())
	} else {
		fmt.Printf("InvokeAction '%s' success. output: %v\n", actionName, output)
	}
}
