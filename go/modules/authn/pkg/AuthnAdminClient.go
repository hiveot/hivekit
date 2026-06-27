// Package authnclient with administration facing methods
package authnpkg

import (
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	authnapi "github.com/hiveot/hivekit/go/modules/authn"
)

// AuthnAdminClient is a client module for authentication management using RRN messages.
// This is a simple wrapper that generates the request messages.
// This should be linked to a transport client module for message delivery.
type AuthnAdminClient struct {
	*modules.HiveModuleBase
	// The ThingID of the authn service that handles the request.
	serviceID string
}

// AddClient adds a new consumer, device or service account.
func (m *AuthnAdminClient) AddClient(clientID string, displayName string, role string, pubKey string) (
	token string, err error) {

	var args = authnapi.AdminAddClientArgs{
		ClientID:    clientID,
		DisplayName: displayName,
		Role:        role,
	}
	err = m.Rpc(td.OpInvokeAction, m.serviceID, authnapi.AdminActionAddClient, &args, &token)
	return
}

// GetClientProfile client method - Get Client Profile.
// Get the profile information describing a client
func (m *AuthnAdminClient) GetClientProfile(clientID string) (
	profile authnapi.ClientProfile, err error) {

	err = m.Rpc(td.OpInvokeAction, m.serviceID,
		authnapi.AdminActionGetProfile, &clientID, &profile)
	return
}

// GetProfiles client method - Get Profiles.
// Get a list of all client profiles
func (m *AuthnAdminClient) GetProfiles() (clientProfiles []authnapi.ClientProfile, err error) {

	err = m.Rpc(td.OpInvokeAction, m.serviceID,
		authnapi.AdminActionGetProfiles, nil, &clientProfiles)
	return
}

// RemoveClient client method - Remove Client.
// Remove a client account
func (m *AuthnAdminClient) RemoveClient(clientID string) (err error) {

	err = m.Rpc(td.OpInvokeAction, m.serviceID,
		authnapi.AdminActionRemoveClient, &clientID, nil)
	return
}

// SetClientPassword client method - Set Client Password.
// Update the password of a consumer
func (m *AuthnAdminClient) SetClientPassword(userName string, password string) (err error) {

	var args = authnapi.AdminSetPasswordArgs{
		UserName: userName, Password: password}

	err = m.Rpc(td.OpInvokeAction, m.serviceID,
		authnapi.AdminActionSetPassword, &args, nil)
	return
}

// UpdateClientProfile client method - Update Client Profile.
// Update the details of a client
func (m *AuthnAdminClient) UpdateClientProfile(clientProfile authnapi.ClientProfile) (err error) {

	err = m.Rpc(td.OpInvokeAction,
		authnapi.AuthnAdminServiceID, m.serviceID, &clientProfile, nil)
	return
}

// Create a new instance of the authentication administration messaging client
//
// sink is the request handler this will forward requests to the authn service.
// This will also set this client as the notification sink for all authn generated notifications.
func NewAuthnAdminClient(sink modules.IHiveModule) *AuthnAdminClient {
	m := &AuthnAdminClient{
		serviceID:      authnapi.DefaultAdminServiceID,
		HiveModuleBase: modules.NewHiveModuleBase("", 0),
	}
	if sink != nil {
		m.SetRequestSink(sink)
		sink.SetNotificationSink(m, m.serviceID)
	}
	return m
}
