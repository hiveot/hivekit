package cliex

import (
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api/td"
)

// Show the content of a remote directory
// This first discovers the directory then attempts to read it.
func (app *Cliex) ListDir() {
	var tdoc *td.TD
	var waitTime = time.Second

	rec0, err := app.discoClient.DiscoverFirstDirectory("", waitTime)
	if err != nil || rec0 == nil {
		fmt.Println("ERROR ListDir: No directory discovered. Need a directory to list")
		return
	}
	tddURL := rec0.AsURL()
	tdoc, _, err = app.discoClient.LoadTD(tddURL)
	if err != nil {
		fmt.Printf(" Error reading TD: %s\n", err.Error())
	}
	if tdoc != nil {
		// for now just show the first 100
		tdList, err := app.dirClient.RetrieveAllThings(0, 100)
		if err != nil {
			fmt.Printf("ERROR: Read directory '%s' failed: %s", tdoc.ID, err.Error())
		} else {
			ListThings(tdList)
		}
	}

}
