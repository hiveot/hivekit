package cliex

import (
	"fmt"

	"github.com/hiveot/hivekit/go/cells/authn"
	authn_client "github.com/hiveot/hivekit/go/cells/authn/client"
)

// List client profile
// If no client ID is given then list all clients
func (app *Cliex) ListAccount(clientID string) error {
	var err error

	// 2. Connect to the authn service
	// the consumer recipe should already have it based on commandline.
	authnAdmin := authn_client.NewAuthnAdminClient(authn.AuthnAdminServiceDefaultThingID, app.co)
	if clientID != "" {
		prof, err := authnAdmin.GetProfile(clientID)
		if err != nil {
			fmt.Println("Failed getting profile: ", err.Error())
			return err
		}
		fmt.Println("Client ID:    ", prof.ClientID)
		fmt.Println("Display Name: ", prof.DisplayName)
		fmt.Println("Role:         ", prof.Role)
		fmt.Println("Created:      ", prof.TimeCreated)
		fmt.Println("Updated:      ", prof.TimeUpdated)
		fmt.Println("Disabled:     ", prof.Disabled)
		return nil
	}
	profList, err := authnAdmin.GetProfiles()
	fmtString := "%-20s %-20s %-10s %-28s %-28s\n"
	fmt.Printf(fmtString, "Client ID", "Display Name", "Role", "Created", "Updated")
	fmt.Printf(fmtString, "---------", "------------", "----", "-------", "-------")
	for _, prof := range profList {
		fmt.Printf(fmtString,
			prof.ClientID, prof.DisplayName, prof.Role, prof.TimeCreated, prof.TimeUpdated)
	}

	return err
}
