package cliex

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/authn"
	authn_client "github.com/hiveot/hivekit/go/cells/authn/client"
	"github.com/hiveot/hivekit/go/cells/certs"
	certs_client "github.com/hiveot/hivekit/go/cells/certs/client"
	"github.com/hiveot/hivekit/go/utils"
)

// Add/update a consumer account with a 30 day auth token in the certs directory.
//
// This requires an admin connection to the gateway or device server.
//
//	clientID is the ID of the device
//	displayName of the device
func (app *Cliex) AddConsumerAccount(clientID string, displayName string) error {

	certsDir := app.env.CertsDir

	// 1. Connect to the authn service
	// the consumer recipe should already have it based on commandline.
	authnAdmin := authn_client.NewAuthnAdminClient(authn.DefaultAuthnAdminThingID, app.co)
	prof, err := authnAdmin.GetProfile(clientID)
	_ = prof

	if err != nil {
		// 2. Client has no account. Create one.
		role := api.ClientRoleViewer
		token, err := authnAdmin.AddClient(clientID, clientID, role, "")
		if err == nil && certsDir != "" {
			tokenFile := filepath.Join(certsDir, clientID+api.DefaultTokenFileSuffix)
			// the old token can't be overwritten so remove it first
			_ = os.Remove(tokenFile)
			err = os.WriteFile(tokenFile, []byte(token), 0400)
		}
		if err == nil {
			fmt.Printf("User account '%s' created successfully\n", clientID)
		} else {
			fmt.Printf("Failed creating a user account '%s': %s\n", clientID, err.Error())
			return err
		}
	} else {
		// update the account token
		err := authnAdmin.UpdateClientProfile(prof)
		if err == nil {
			fmt.Printf("User account '%s' updated successfully\n", clientID)
		} else {
			fmt.Printf("Failed updating the device account for client '%s': %s\n", clientID, err.Error())
			return err
		}
		newToken, err := authnAdmin.CreateToken(clientID)
		if err == nil {
			tokenFile := filepath.Join(certsDir, clientID+api.DefaultTokenFileSuffix)
			// the old token can't be overwritten so remove it first
			_ = os.Remove(tokenFile)
			err = os.WriteFile(tokenFile, []byte(newToken), 0400)
		}
		if err == nil {
			fmt.Printf("Auth token for '%s' refreshed successfully\n", clientID)
		} else {
			fmt.Printf("Failed updating the auth token '%s': %s\n", clientID, err.Error())
			return err
		}
	}

	return err
}

// Add/update a device account with a 30 day client certificate in the certs directory.
//
// This requires an admin connection to the gateway or device server.
//
//	clientID is the ID of the device
//	displayName of the device
func (app *Cliex) AddDeviceAccount(clientID string, displayName string) error {

	certsDir := app.env.CertsDir

	// get or create the device private/public key for the account and cert
	certPath := filepath.Join(app.env.CertsDir, clientID+api.DefaultCertFileSuffix)
	keyPath := filepath.Join(app.env.CertsDir, clientID+api.DefaultKeyFileSuffix)
	privKey, err := utils.LoadCreateKeyPair(keyPath, utils.KeyTypeED25519)
	if err != nil {
		return err
	}

	// 2. Connect to the authn service
	// the consumer recipe should already have it based on commandline.
	authnAdmin := authn_client.NewAuthnAdminClient(authn.DefaultAuthnAdminThingID, app.co)
	prof, err := authnAdmin.GetProfile(clientID)
	_ = prof

	if err != nil {
		// 3. Client has no account. Create one.
		pubKeyPEM := utils.PublicKeyToPem(privKey.Public())
		token, err := authnAdmin.AddClient(
			clientID, clientID, api.ClientRoleDevice, pubKeyPEM)
		if err == nil && certsDir != "" {
			tokenFile := filepath.Join(certsDir, clientID+api.DefaultTokenFileSuffix)
			// the old token can't be overwritten
			_ = os.Remove(tokenFile)
			err = os.WriteFile(tokenFile, []byte(token), 0400)
		}
		if err == nil {
			fmt.Printf("Device account for client '%s' created successfully\n", clientID)
		} else {
			fmt.Printf("Failed creating a device account for client '%s': %s\n", clientID, err.Error())
			return err
		}
	} else {
		// update the device token
		err := authnAdmin.UpdateClientProfile(prof)
		if err == nil {
			fmt.Printf("Device account for client '%s' updated successfully\n", clientID)
		} else {
			fmt.Printf("Failed updating the device account for client '%s': %s\n", clientID, err.Error())
			return err
		}
	}

	// 3. Create and save the TLS cert
	if certsDir != "" {
		certsCl := certs_client.NewCertsClient(certs.CertsServiceDefaultThingID, app.co)
		x509Cert, err := certsCl.CreateClientCert(
			clientID, api.ClientRoleDevice, 0, privKey.Public())

		if err == nil {
			tlsCert := utils.X509CertToTLS(x509Cert, privKey)
			err = utils.SaveTLSCert(tlsCert, certPath, keyPath)
		}
		if err == nil {
			fmt.Printf("New client certificate has been successfully created in %s\n", certPath)
		} else {
			fmt.Printf("Failed updating the client certificate: %s\n", err.Error())
		}
	}

	return err
}
