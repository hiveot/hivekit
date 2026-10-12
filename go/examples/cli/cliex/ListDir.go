package cliex

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/discovery"
)

// Show the content of a remote directory.
// if a thingID is not provided, discover it first
// This first discovers the directory then attempts to read it.
func (app *Cliex) ListDir(env *api.HiveEnvironment, thingID string) {
	var waitTime = time.Second
	var err error

	tdURL := env.DirTDURL
	dirTD := env.GetDirTD()
	if dirTD == nil && tdURL != "" {
		dirTD, _, err = app.discoClient.LoadTD(tdURL)
		if err != nil {
			slog.Error("ListDir: Failed downloading the TD using the provided URL", "tdURL",
				tdURL, "err", err.Error())
			return
		}
		if !dirTD.IsDirectory() {
			slog.Error("ListDir: The downloaded TD is not a directory TD", "tdURL",
				tdURL, "err", err.Error())
			return
		}
	}

	// Attempt to discover a directory
	if dirTD == nil {
		dirTD = app.discoClient.DiscoverFirstTD(thingID, discovery.DISCO_TYPE_DIRECTORY, waitTime)
	}

	if dirTD == nil {
		fmt.Println("ERROR ListDir: No directory discovered. Need a directory to list")
		return
	}
	fmt.Printf("Found directory with thingID '%s'\n", dirTD.ID)

	// for now just show up to the first 100 entries
	app.dirClient.SetTDD(dirTD)
	tdList, cacheOnly := app.dirClient.RetrieveAllThings(0, 100)
	if cacheOnly {
		ListThings(tdList)
		fmt.Printf("ERROR: Read directory '%s' failed. Used cached TDs\n", dirTD.ID)
	} else {
		ListThings(tdList)
		fmt.Printf("Directory '%s' contains %d Things\n", dirTD.ID, len(tdList))
	}

}
