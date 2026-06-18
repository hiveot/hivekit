package wotcli

import (
	"fmt"
	"io"
	"net/http"

	"github.com/araddon/dateparse"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/examples/wotco"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
)

// Show a list of Thing TDs
func ListThings(tdList []*td.TD) {
	fmt.Printf("Thing ID                     Title                         #Props #Events #Actions  Modified (local)  base\n")
	fmt.Printf("---------------------------  ----------------------------  ------ ------- --------  ----------------  -----\n")
	for _, tdoc := range tdList {
		modified := dateparse.MustParse(tdoc.Modified).Local()

		fmt.Printf("%-28s %-28.28s %6d %7d %8d   %-16s  %-20s\n",
			tdoc.ID, tdoc.Title, len(tdoc.Properties), len(tdoc.Events), len(tdoc.Actions),
			modified.Format("2006-01-02 15:04"), tdoc.Base)
	}
}

// discover things and directories on the network
//
// if readDir is true then try to read the directory content/hivekit/go/examples/wotco"
func ShowDiscovery(co *wotco.WotConsumer, verbose bool) {
	nrFound := 0

	fmt.Println("Discovered Things and Directories on the local network")
	fmt.Printf("Type       Address    Port   Instance Name        Schema   ThingID                           TD URL   \n")
	fmt.Printf("---------- ---------- -----  -------------------  -------  --------------------------------  -------  \n")

	co.Discover(func(r *discovery.DiscoveryResult) bool {
		// load the TD to present nr of affordances
		tdURL := r.AsURL()
		var tdoc *td.TD
		var thingID string = "n/a"
		if tdURL != "" {
			resp, err := http.Get(tdURL)
			if err == nil {
				raw, _ := io.ReadAll(resp.Body)
				tdoc, err = td.UnmarshalTD(string(raw))
			}
			if err != nil {
				fmt.Printf(" Error reading TD: %s\n", err.Error())
			} else {
				thingID = tdoc.ID
			}
		}
		// show the discovery record and the nr of affordances in the TD
		fmt.Printf("%-10s %-10s %-5d  %-20s %-8s %-33s %s \n",
			r.Type, r.Addr, r.Port, r.Instance, r.Schema, thingID, tdURL)

		if verbose {
			fmt.Printf("Thing ID: %s\n", tdoc.ID)
			fmt.Printf("Base: %s\n", tdoc.Base)
			fmt.Printf("  Affordance   Name                        Title\n")
			fmt.Printf("  ----------   ----                        -----\n")
			for k, v := range tdoc.Properties {
				fmt.Printf("  property     %-26s  %s\n", k, v.Title)
			}
			for k, v := range tdoc.Events {
				fmt.Printf("  event        %-26s  %s\n", k, v.Title)
			}
			for k, v := range tdoc.Actions {
				fmt.Printf("  action       %-26s  %s\n", k, v.Title)
			}

			// show TXT record
			fmt.Printf("  TXT Records\n")
			for k, v := range r.Params {
				fmt.Printf("  %10s: %s\n", k, v)
			}
		}

		fmt.Println()
		nrFound++
		return false
	})
	fmt.Printf("Found %d records\n", nrFound)
}
