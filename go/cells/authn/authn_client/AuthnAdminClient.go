// Package authnclient with administration facing methods
package authn_client

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells"
	authnapi "github.com/hiveot/hivekit/go/cells/authn"
)

// AuthnAdminClient is a client for authentication management using RRN messages.
// This is a simple wrapper that generates the request messages.
// This should be linked to a transport client for message delivery.
type AuthnAdminClient struct {
	*cells.HiveCellBase
	// The instance ID of the authn admin service that handles the request.
	authnServiceID string
}

// AddClient adds a new consumer, device or service account.
func (cl *AuthnAdminClient) AddClient(clientID string, displayName string, role string, pubKey string) (
	token string, err error) {

	var args = authnapi.AdminAddClientInput{
		ClientID:    clientID,
		DisplayName: displayName,
		Role:        role,
	}
	err = cl.Rpc(wot.OpInvokeAction, cl.authnServiceID,
		authnapi.AdminAddClientAction, &args, &token)
	return
}

// GetClientProfile client method - Get Client Profile.
// Get the profile information describing a client
func (cl *AuthnAdminClient) GetClientProfile(clientID string) (
	profile authnapi.ClientProfile, err error) {

	err = cl.Rpc(wot.OpInvokeAction, cl.authnServiceID,
		authnapi.AdminGetProfileAction, &clientID, &profile)
	return
}

// GetProfiles client method - Get Profiles.
// Get a list of all client profiles
func (cl *AuthnAdminClient) GetProfiles() (clientProfiles []authnapi.ClientProfile, err error) {

	err = cl.Rpc(wot.OpInvokeAction, cl.authnServiceID,
		authnapi.AdminGetProfilesAction, nil, &clientProfiles)
	return
}

// RemoveClient client method - Remove Client.
// Remove a client account
func (cl *AuthnAdminClient) RemoveClient(clientID string) (err error) {

	err = cl.Rpc(wot.OpInvokeAction, cl.authnServiceID,
		authnapi.AdminRemoveClientAction, &clientID, nil)
	return
}

// SetClientPassword client method - Set Client Password.
// Update the password of a consumer
func (cl *AuthnAdminClient) SetClientPassword(userName string, password string) (err error) {

	var args = authnapi.AdminSetPasswordInput{
		UserName: userName, Password: password}

	err = cl.Rpc(wot.OpInvokeAction, cl.authnServiceID,
		authnapi.AdminSetPasswordAction, &args, nil)
	return
}

// UpdateClientProfile client method - Update Client Profile.
// Update the details of a client
func (cl *AuthnAdminClient) UpdateClientProfile(clientProfile authnapi.ClientProfile) (err error) {

	err = cl.Rpc(wot.OpInvokeAction, cl.authnServiceID,
		authnapi.AdminUpdateProfileAction, &clientProfile, nil)
	return
}

// Create and link a new instance of the authentication administration messaging client
//
//	authnServiceID is the authn admin service instance ID
//	sink is the optional request handler this will forward requests to the authn service.
//
// This will also set this client as the notification sink for all authn generated notifications.
func NewAuthnAdminClient(authnServiceID string, sink api.IHiveCell) *AuthnAdminClient {
	cl := &AuthnAdminClient{
		HiveCellBase:   cells.NewHiveCellBase(""),
		authnServiceID: authnServiceID,
	}
	if sink != nil {
		cl.SetRequestSink(sink)
		sink.SetNotificationSink(cl, cl.authnServiceID)
	}
	return cl
}
