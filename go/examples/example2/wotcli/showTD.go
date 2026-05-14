package wotcli

import (
	"fmt"
	"io"
	"net/http"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/examples/wotco"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transports/discovery/pkg"
	jsoniter "github.com/json-iterator/go"
)

// Display the TD of a discovery record by its instance
func ShowTD(co *wotco.WotConsumer, discoInstance string) {

	found := false
	co.Discover(func(r *discoverypkg.DiscoveryResult) bool {
		tdURL := r.AsURL()
		var tdoc *td.TD
		if r.Instance != discoInstance {
			return false // keep looking
		}
		if tdURL == "" {
			fmt.Println("No TD URL found in Discovery Record with Instance: " + r.Instance)
			return true
		}
		resp, err := http.Get(tdURL)
		if err != nil {
			fmt.Printf("\nError while retrieving TD from URL: %s\n", tdURL)
			fmt.Printf("Error Message: %s\n", err)
			return true
		}

		raw, _ := io.ReadAll(resp.Body)
		tdoc, _ = td.UnmarshalTD(string(raw))

		pretty, err := jsoniter.MarshalIndent(tdoc, "", "  ")
		_ = err
		fmt.Println("Found discovery record with Instance name: " + discoInstance)
		fmt.Println("Discovery record points to TD with ID: " + tdoc.ID)
		fmt.Println(string(pretty))
		found = true
		// done
		return true
	})
	if !found {
		fmt.Printf("\nNo Discovery Record found that matches the given Instance: %s\n", discoInstance)
	}
}
