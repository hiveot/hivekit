package wotcli

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api/td"
)

// Show the content of a remote directory
// This first discovers the directory then attempts to read it.
func (app *CliApp) ListDir() {
	var tdoc *td.TD
	var waitTime = time.Second

	rec0, err := app.discoClient.DiscoverFirstDirectory("", waitTime)
	if err != nil {
		slog.Error("ListDir: Discovery failed")
		return
	} else if rec0 == nil {
		fmt.Println("ListDir: No directory found")
		return
	}
	tddURL := rec0.AsURL()
	tdoc, _, err = app.discoClient.LoadTD(tddURL, app.caCert)
	if err != nil {
		fmt.Printf(" Error reading TD: %s\n", err.Error())
	}
	if tdoc != nil {
		// for now just show the first 100
		tdList, err := app.dirClient.RetrieveAllThings(0, 100)
		if err != nil {
			fmt.Printf("Read directory '%s' failed: %s", tdoc.ID, err.Error())
		} else {
			ListThings(tdList)
		}
	}

}
