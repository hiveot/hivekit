package cliex

import (
	"fmt"
)

// Show the actions of a Thing
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

	// 3. check for invoking an action
	if actionName != "" && tdoc.Actions[actionName] != nil {
		// invoke the action
		println("Invoking action: ", actionName)
		err := app.co.InvokeAction(thingID, actionName, nil, nil)
		if err != nil {
			fmt.Printf("InvokeAction '%s' returned error: %s", actionName, err.Error())
		}
	} else {
		// show the action
		println("Actions:")
		for k, aff := range tdoc.Actions {
			fmt.Printf("  %s: %v\n", k, aff.Title)
		}
	}
}
