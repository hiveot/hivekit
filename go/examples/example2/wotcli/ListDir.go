package wotcli

import (
	"fmt"
	"io"
	"net/http"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/examples/wotco"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transport/discovery/pkg"
)

func ListDir(co *wotco.WotConsumer) {
	var dirTD string
	var tdoc *td.TD

	co.Discover(func(r *discoverypkg.DiscoveryResult) bool {

		if r.IsDirectory {
			tdURL := r.AsURL()
			resp, err := http.Get(tdURL)
			if err != nil {
				fmt.Printf(" Error reading TD: %s\n", err.Error())
				return false // continue
			}
			raw, _ := io.ReadAll(resp.Body)
			dirTD = string(raw)
			return true // done
		}
		return false
	})
	if dirTD != "" {
		tdoc, _ = td.UnmarshalTD(dirTD)

		tdList, err := co.ReadDirectory(tdoc, 100)
		if err != nil {
			fmt.Printf("Read directory '%s' failed: %s", tdoc.ID, err.Error())
		} else {
			ListThings(tdList)
		}
	}

}
