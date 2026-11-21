// Package authnclient with administration facing methods
package authn_client

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/authn"
)

// AuthnAdminClient is a client for authentication management using RRN messages.
// This is a simple wrapper that generates the request messages.
// This should be linked to a transport client for message delivery.
type AuthnAdminClient struct {
	*cells.HiveCellBase
	// The instance ID of the authn admin service that handles the request.
	authnAdminServiceID string
}

// AddClient add a new client account. This fails if the client already exists.
//
// Use session authenticator's SetPassword or CreateToken to obtain a session
// token to connect with.
//
//	clientID is the account ID of the client
//	displayName is the friendly name of the client
//	role is the client role, eg ClientRoleViewer, ... ClientRoleDevice
func (cl *AuthnAdminClient) AddClient(clientID string, displayName string, role string) (
	token string, err error) {

	var args = authn.AdminAddClientInput{
		ClientID:    clientID,
		DisplayName: displayName,
		Role:        role,
	}
	if displayName == "" {
		displayName = clientID
	}
	err = cl.Rpc(wot.OpInvokeAction, cl.authnAdminServiceID,
		authn.AdminAddClientAction, &args, &token)
	return token, err
}

// CreateToken creates a new client authentication token with the default expiry based on the role
func (cl *AuthnAdminClient) CreateToken(clientID string) (newToken string, err error) {

	err = cl.Rpc(wot.OpInvokeAction, cl.authnAdminServiceID,
		authn.AdminCreateTokenAction, &clientID, &newToken)
	return
}

// GetProfile client method - Get Client Profile.
// Get the profile information describing a client
func (cl *AuthnAdminClient) GetProfile(clientID string) (
	profile authn.ClientProfile, err error) {

	err = cl.Rpc(wot.OpInvokeAction, cl.authnAdminServiceID,
		authn.AdminGetProfileAction, &clientID, &profile)
	return profile, err
}

// GetProfiles client method - Get Profiles.
// Get a list of all client profiles
func (cl *AuthnAdminClient) GetProfiles() (clientProfiles []authn.ClientProfile, err error) {

	err = cl.Rpc(wot.OpInvokeAction, cl.authnAdminServiceID,
		authn.AdminGetProfilesAction, nil, &clientProfiles)
	return clientProfiles, err
}

// RemoveClient client method - Remove Client.
// Remove a client account
func (cl *AuthnAdminClient) RemoveClient(clientID string) (err error) {

	err = cl.Rpc(wot.OpInvokeAction, cl.authnAdminServiceID,
		authn.AdminRemoveClientAction, &clientID, nil)
	return err
}

// SetClientPassword client method - Set Client Password.
// Update the password of a consumer
func (cl *AuthnAdminClient) SetClientPassword(userName string, password string) (err error) {

	var args = authn.AdminSetPasswordInput{
		UserName: userName, Password: password}

	err = cl.Rpc(wot.OpInvokeAction, cl.authnAdminServiceID,
		authn.AdminSetPasswordAction, &args, nil)
	return err
}

// SetRole - Set client role.
//
//	clientID of the client whose role to update
//	role to set, api.ClientRoleXyz
func (cl *AuthnAdminClient) SetRole(clientID string, role string) (err error) {

	var args = authn.AdminSetRoleInput{
		ClientID: clientID, Role: role}

	err = cl.Rpc(wot.OpInvokeAction, cl.authnAdminServiceID,
		authn.AdminSetRoleAction, &args, nil)
	return err
}

// UpdateClientProfile client method - Update Client Profile.
// Update the details of a client
func (cl *AuthnAdminClient) UpdateClientProfile(clientProfile authn.ClientProfile) (err error) {

	err = cl.Rpc(wot.OpInvokeAction, cl.authnAdminServiceID,
		authn.AdminUpdateProfileAction, &clientProfile, nil)
	return err
}

// Create and link a new instance of the authentication administration messaging client
//
// If a 'linkTo' sink is provided it will be used to send requests to and receive notifications from.
// This sets the notification sink for authn-admin notifications to this client.
//
//	authnServiceID is the authn admin service instance ID
//	linkTo is the chain that forwards requests and returns notifications.
//
// This will also set this client as the notification sink for all authn generated notifications.
func NewAuthnAdminClient(authnServiceID string, linkTo api.IHiveCell) *AuthnAdminClient {
	if authnServiceID == "" {
		authnServiceID = authn.AuthnAdminServiceCellType
	}
	cl := &AuthnAdminClient{
		HiveCellBase:        cells.NewHiveCellBase(authn.AuthnAdminServiceCellType + "-client"),
		authnAdminServiceID: authnServiceID,
	}
	if linkTo != nil {
		cl.SetRequestSink(linkTo)
		linkTo.SetNotificationSink(cl)
	}
	return cl
}
