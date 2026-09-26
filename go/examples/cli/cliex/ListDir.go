package cliex

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
)

// Show the content of a remote directory.
// if a thingID is not provided, discover it first
// This first discovers the directory then attempts to read it.
// FIXME: listDir should use the provided environment tddURL if provided
func (app *Cliex) ListDir(env *api.HiveEnvironment, thingID string) {
	var waitTime = time.Second
	var err error

	tdURL := env.ServerTDURL
	serverTD := env.ServerTD
	if serverTD == nil && tdURL != "" {
		serverTD, _, err = app.discoClient.LoadTD(tdURL)
		if err != nil {
			slog.Error("ListDir: Failed downloading the TD using the provided URL", "tdURL",
				tdURL, "err", err.Error())
			return
		}
		if !serverTD.IsDirectory() {
			slog.Error("ListDir: The downloaded TD is not a directory TD", "tdURL",
				tdURL, "err", err.Error())
			return
		}
	}

	// Attempt to discover a directory
	if serverTD == nil {
		serverTD = app.discoClient.DiscoverFirstTD(thingID, discovery.DISCO_TYPE_DIRECTORY, waitTime)
	}

	if serverTD == nil {
		fmt.Println("ERROR ListDir: No directory discovered. Need a directory to list")
		return
	}
	fmt.Printf("Found directory with thingID '%s'\n", serverTD.ID)

	// for now just show up to the first 100 entries
	app.dirClient.SetTDD(serverTD)
	tdList, err := app.dirClient.RetrieveAllThings(0, 100)
	if err != nil {
		fmt.Printf("ERROR: Read directory '%s' failed: %s\n", serverTD.ID, err.Error())
	} else {
		ListThings(tdList)
		fmt.Printf("Directory '%s' contains %d Things\n", serverTD.ID, len(tdList))
	}

}
