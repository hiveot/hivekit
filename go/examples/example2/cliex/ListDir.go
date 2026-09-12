package cliex

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api"
)

// Show the content of a remote directory.
// if a thingID is not provided, discover it first
// This first discovers the directory then attempts to read it.
// FIXME: listDir should use the provided environment tddURL if provided
func (app *Cliex) ListDir(env *api.HiveEnvironment, thingID string) {
	var waitTime = time.Second
	var err error

	tddURL := env.TDDURL
	dirTD := env.DirTD
	if dirTD == nil && tddURL != "" {
		dirTD, _, err = app.discoClient.LoadTD(tddURL)
		if err != nil {
			slog.Error("ListDir: Failed downloading the TDD using the provided TD URL", "tddURL",
				tddURL, "err", err.Error())
			return
		}
	}

	// Attempt to discover a directory
	if dirTD == nil {
		dirTD, tddURL, _, err = app.discoClient.DiscoverFirstDirectoryTD(thingID, waitTime)
		_ = tddURL
	}

	if err != nil || dirTD == nil {
		fmt.Println("ERROR ListDir: No directory discovered. Need a directory to list")
		return
	}
	fmt.Printf("Found directory with thingID '%s'\n", dirTD.ID)

	// for now just show up to the first 100 entries
	app.dirClient.SetTDD(dirTD)
	tdList, err := app.dirClient.RetrieveAllThings(0, 100)
	if err != nil {
		fmt.Printf("ERROR: Read directory '%s' failed: %s\n", dirTD.ID, err.Error())
	} else {
		ListThings(tdList)
		fmt.Printf("Directory '%s' contains %d Things\n", dirTD.ID, len(tdList))
	}

}
