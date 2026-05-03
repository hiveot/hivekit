package discover

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hiveot/hivekit/go/api/td"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transports/discovery/pkg"
	jsoniter "github.com/json-iterator/go"
)

// discover things and directories on the network
func Discover(filterType string, filterAddr string, showAff, showTXT, showTD bool, waitSec int) {
	nrFound := 0
	filterType = strings.ToLower(filterType)
	filterAddr = strings.ToLower(filterAddr)

	disco := discoverypkg.NewDiscoveryClient()
	fmt.Println("Discovered Things and Directories on the local network")
	fmt.Printf("Type       Address    Port   Instance             Schema    nrProps   nrEvents  nrActions   TD URL   \n")
	fmt.Printf("---------- ---------- -----  -------------------  -------   -------   --------  ---------   -------  \n")
	waitDuration := time.Duration(waitSec) * time.Second

	disco.DiscoverThings("", waitDuration, func(r *discoverypkg.DiscoveryResult) bool {
		var nrProps = 0
		var nrEvents = 0
		var nrActions = 0

		if filterType != "" && filterType != strings.ToLower(r.Type) {
			return false
		}
		if filterAddr != "" && filterAddr != strings.ToLower(r.Addr) {
			return false
		}

		// load the TD to present nr of affordances
		tdURL := r.AsURL()
		var tdoc *td.TD
		if tdURL != "" {
			resp, err := http.Get(tdURL)
			if err == nil {
				raw, _ := io.ReadAll(resp.Body)
				tdoc, _ = td.UnmarshalTD(string(raw))
				nrProps = len(tdoc.Properties)
				nrEvents = len(tdoc.Events)
				nrActions = len(tdoc.Actions)
			}
			if err != nil {
				fmt.Printf(" Error reading TD: %s\n", err.Error())
			}
		}

		fmt.Printf("%-10s %-10s %-5d  %-20s %-8s      %3d        %3d        %3d   %s \n",
			r.Type, r.Addr, r.Port, r.Instance, r.Schema, nrProps, nrEvents, nrActions, tdURL)

		if showAff {
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
		}
		if showTXT {
			for k, v := range r.Params {
				fmt.Printf("  %10s: %s\n", k, v)
			}
		}
		if showTD {
			pretty, err := jsoniter.MarshalIndent(tdoc, "", "  ")
			_ = err
			fmt.Println(string(pretty))
		}
		fmt.Println()
		nrFound++
		return false
	})
	fmt.Printf("Found %d records\n", nrFound)
}
