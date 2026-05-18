package wotcli

import (
	"fmt"

	"github.com/hiveot/hivekit/go/examples/wotco"
	jsoniter "github.com/json-iterator/go"
)

// Display the TD of a discovered Thing
func ShowTD(co *wotco.WotConsumer, thingID string) {

	fmt.Printf("Running discovery...   ")
	err := co.Discover(nil)
	if err != nil {
		fmt.Printf("Discovery failed: %v\n", err)
		return
	}
	fmt.Printf("found %d Thing(s)", len(co.GetThings()))
	if len(co.GetDirectories()) > 0 {
		fmt.Printf(" and %d directory(ies)\n", len(co.GetDirectories()))
	} else {
		fmt.Println(" and no directories")
	}

	tdoc := co.GetTD(thingID)
	if tdoc == nil {
		fmt.Printf("TD for Thing '%s' not discovered\n", thingID)
		return
	}
	fmt.Printf("Showing TD for Thing '%s':\n", tdoc.ID)
	pretty, err := jsoniter.MarshalIndent(tdoc, "", "  ")
	_ = err
	fmt.Println(string(pretty))
}
