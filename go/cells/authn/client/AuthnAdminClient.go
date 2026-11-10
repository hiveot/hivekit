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
	authnServiceID string
}

// AddClient add a new client account. This fails if the client already exists.
//
// Use session authenticator's SetPassword or CreateToken to obtain a session
// token to connect with.
//
//	clientID is the account ID of the client
//	displayName is the friendly name of the client
//	role is the client role, eg ClientRoleViewer, ... ClientRoleDevice
func (cl *AuthnAdminClient) AddClient(clientID string, displayName string, role string, pubKey string) (
	token string, err error) {

	var args = authn.AdminAddClientInput{
		ClientID:    clientID,
		DisplayName: displayName,
		Role:        role,
	}
	err = cl.Rpc(wot.OpInvokeAction, cl.authnServiceID,
		authn.AdminAddClientAction, &args, &token)
	return
}

// CreateToken creates a new client authentication token with the default expiry based on the role
func (cl *AuthnAdminClient) CreateToken(clientID string) (newToken string, err error) {

	err = cl.Rpc(wot.OpInvokeAction, cl.authnServiceID,
		authn.AdminCreateTokenAction, &clientID, &newToken)
	return
}

// GetProfile client method - Get Client Profile.
// Get the profile information describing a client
func (cl *AuthnAdminClient) GetProfile(clientID string) (
	profile authn.ClientProfile, err error) {

	err = cl.Rpc(wot.OpInvokeAction, cl.authnServiceID,
		authn.AdminGetProfileAction, &clientID, &profile)
	return
}

// GetProfiles client method - Get Profiles.
// Get a list of all client profiles
func (cl *AuthnAdminClient) GetProfiles() (clientProfiles []authn.ClientProfile, err error) {

	err = cl.Rpc(wot.OpInvokeAction, cl.authnServiceID,
		authn.AdminGetProfilesAction, nil, &clientProfiles)
	return
}

// RemoveClient client method - Remove Client.
// Remove a client account
func (cl *AuthnAdminClient) RemoveClient(clientID string) (err error) {

	err = cl.Rpc(wot.OpInvokeAction, cl.authnServiceID,
		authn.AdminRemoveClientAction, &clientID, nil)
	return
}

// SetClientPassword client method - Set Client Password.
// Update the password of a consumer
func (cl *AuthnAdminClient) SetClientPassword(userName string, password string) (err error) {

	var args = authn.AdminSetPasswordInput{
		UserName: userName, Password: password}

	err = cl.Rpc(wot.OpInvokeAction, cl.authnServiceID,
		authn.AdminSetPasswordAction, &args, nil)
	return
}

// UpdateClientProfile client method - Update Client Profile.
// Update the details of a client
func (cl *AuthnAdminClient) UpdateClientProfile(clientProfile authn.ClientProfile) (err error) {

	err = cl.Rpc(wot.OpInvokeAction, cl.authnServiceID,
		authn.AdminUpdateProfileAction, &clientProfile, nil)
	return
}

// Create and link a new instance of the authentication administration messaging client
//
//	authnServiceID is the authn admin service instance ID
//	sink is the optional request handler this will forward requests to the authn service.
//
// This will also set this client as the notification sink for all authn generated notifications.
func NewAuthnAdminClient(authnServiceID string, sink api.IHiveCell) *AuthnAdminClient {
	if authnServiceID == "" {
		authnServiceID = authn.DefaultAuthnAdminThingID
	}
	cl := &AuthnAdminClient{
		HiveCellBase:   cells.NewHiveCellBase(authn.AuthnAdminServiceType + "-client"),
		authnServiceID: authnServiceID,
	}
	if sink != nil {
		cl.SetRequestSink(sink)
		sink.SetNotificationSink(cl, cl.authnServiceID)
	}
	return cl
}
