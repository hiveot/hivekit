// Package authnclient with administration facing methods
package authnclient

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	authnapi "github.com/hiveot/hivekit/go/cells/authn"
)

// AuthnAdminClient is a client for authentication management using RRN messages.
// This is a simple wrapper that generates the request messages.
// This should be linked to a transport client for message delivery.
type AuthnAdminClient struct {
	*cells.HiveCellBase
	// The ThingID of the authn service that handles the request.
	serviceID string
}

// AddClient adds a new consumer, device or service account.
func (cl *AuthnAdminClient) AddClient(clientID string, displayName string, role string, pubKey string) (
	token string, err error) {

	var args = authnapi.AdminAddClientArgs{
		ClientID:    clientID,
		DisplayName: displayName,
		Role:        role,
	}
	err = cl.Rpc(td.OpInvokeAction, cl.serviceID, authnapi.AdminActionAddClient, &args, &token)
	return
}

// GetClientProfile client method - Get Client Profile.
// Get the profile information describing a client
func (cl *AuthnAdminClient) GetClientProfile(clientID string) (
	profile authnapi.ClientProfile, err error) {

	err = cl.Rpc(td.OpInvokeAction, cl.serviceID,
		authnapi.AdminActionGetProfile, &clientID, &profile)
	return
}

// GetProfiles client method - Get Profiles.
// Get a list of all client profiles
func (cl *AuthnAdminClient) GetProfiles() (clientProfiles []authnapi.ClientProfile, err error) {

	err = cl.Rpc(td.OpInvokeAction, cl.serviceID,
		authnapi.AdminActionGetProfiles, nil, &clientProfiles)
	return
}

// RemoveClient client method - Remove Client.
// Remove a client account
func (cl *AuthnAdminClient) RemoveClient(clientID string) (err error) {

	err = cl.Rpc(td.OpInvokeAction, cl.serviceID,
		authnapi.AdminActionRemoveClient, &clientID, nil)
	return
}

// SetClientPassword client method - Set Client Password.
// Update the password of a consumer
func (cl *AuthnAdminClient) SetClientPassword(userName string, password string) (err error) {

	var args = authnapi.AdminSetPasswordArgs{
		UserName: userName, Password: password}

	err = cl.Rpc(td.OpInvokeAction, cl.serviceID,
		authnapi.AdminActionSetPassword, &args, nil)
	return
}

// UpdateClientProfile client method - Update Client Profile.
// Update the details of a client
func (cl *AuthnAdminClient) UpdateClientProfile(clientProfile authnapi.ClientProfile) (err error) {

	err = cl.Rpc(td.OpInvokeAction,
		authnapi.AuthnAdminServiceID, cl.serviceID, &clientProfile, nil)
	return
}

// Start and link a new instance of the authentication administration messaging client
//
// sink is the optional request handler this will forward requests to the authn service.
// This will also set this client as the notification sink for all authn generated notifications.
func StartAuthnAdminClient(sink api.IHiveCell) *AuthnAdminClient {
	m := &AuthnAdminClient{
		serviceID:    authnapi.DefaultAdminServiceID,
		HiveCellBase: cells.NewHiveCellBase("", 0),
	}
	if sink != nil {
		m.SetRequestSink(sink)
		sink.SetNotificationSink(m, m.serviceID)
	}
	return m
}
