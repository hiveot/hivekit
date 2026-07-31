package cliex

import (
	"fmt"
	"time"
)

// Show the content of a remote directory
// if a thingID is not provided, discover it first
// This first discovers the directory then attempts to read it.
func (app *Cliex) ListDir(thingID string) {
	var waitTime = time.Second

	dirTD, rec0, err := app.discoClient.DiscoverFirstDirectoryTD(thingID, waitTime)
	_ = rec0

	if err != nil || dirTD == nil {
		fmt.Println("ERROR ListDir: No directory discovered. Need a directory to list")
		return
	} else {
		fmt.Printf("Found directory with thingID '%s'\n", dirTD.ID)
	}
	// for now just show the first 100
	app.dirClient.SetTDD(dirTD)
	tdList, err := app.dirClient.RetrieveAllThings(0, 100)
	if err != nil {
		fmt.Printf("ERROR: Read directory '%s' failed: %s\n", dirTD.ID, err.Error())
	} else {
		ListThings(tdList)
		fmt.Printf("Found %d Things\n", len(tdList))
	}

}
