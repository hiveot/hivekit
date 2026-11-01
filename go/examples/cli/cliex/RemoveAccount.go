package cliex

import (
	"fmt"

	"github.com/hiveot/hivekit/go/cells/authn"
	authn_client "github.com/hiveot/hivekit/go/cells/authn/client"
)

func (app *Cliex) RemoveAccount(clientID string) error {
	var err error

	// 2. Connect to the authn service
	// the consumer recipe should already have it based on commandline.
	authnAdmin := authn_client.NewAuthnAdminClient(authn.DefaultAuthnAdminThingID, app.co)

	_, err = authnAdmin.GetProfile(clientID)
	if err != nil {
		fmt.Println("No account for client", clientID)
		return err
	}
	err = authnAdmin.RemoveClient(clientID)
	if err != nil {
		fmt.Println("Remove account failed: ", err.Error())
	} else {
		fmt.Printf("Account for '%s' succesfully removed\n", clientID)
	}

	return err
}
