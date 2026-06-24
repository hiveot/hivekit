package wotcli

import (
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api/td"
	jsoniter "github.com/json-iterator/go"
)

// Display the TD of a discovered Thing
func (app *CliApp) ShowTD(thingID string) {
	var tdoc *td.TD
	var waitTime = time.Second

	// need the TDD for the directory client to connect to the directory server.
	fmt.Printf("Running discovery...   ")
	rec0, err := app.discoClient.DiscoverFirstDirectory("", waitTime)
	if err != nil {
		fmt.Printf("Discovery failed: %v\n", err)
		return
	}

	dirURL := rec0.AsURL()
	tdd, _, err := app.discoClient.LoadTD(dirURL, app.caCert)
	app.dirClient.SetTDD(tdd)

	// fmt.Printf("found %d Thing(s)", len(tdList))
	// if len(app.co.GetDirectories()) > 0 {
	// fmt.Printf(" and %d directory(ies)\n", len(co.GetDirectories()))
	// } else {
	// fmt.Println(" and no directories")
	// }

	tdoc, err = app.dirClient.RetrieveThing(thingID)
	if tdoc == nil {
		fmt.Printf("TD for Thing '%s' not discovered\n", thingID)
		return
	}
	fmt.Printf("Showing TD for Thing '%s':\n", tdoc.ID)
	pretty, err := jsoniter.MarshalIndent(tdoc, "", "  ")
	_ = err
	fmt.Println(string(pretty))
}
