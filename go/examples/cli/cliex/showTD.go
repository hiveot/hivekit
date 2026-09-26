package cliex

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
)

// Display the TD of a discovered Thing
func (app *Cliex) ShowTD(thingID string) {

	tdoc := app.FindTD(thingID)
	if tdoc == nil {
		fmt.Println("ShowTD TD for thing not found")
		return
	}

	fmt.Printf("Showing TD for Thing '%s':\n", tdoc.ID)
	pretty, err := jsoniter.MarshalIndent(tdoc, "", "  ")
	_ = err
	fmt.Println(string(pretty))
}
