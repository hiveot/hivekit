// Package authnclient with administration facing methods
package authnpkg

import (
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	authnapi "github.com/hiveot/hivekit/go/modules/authn"
)

// AuthnAdminMsgClient is a client module for authentication management using RRN messages.
// This should be linked to a transport client module for message delivery.
type AuthnAdminMsgClient struct {
	*modules.HiveModuleBase
	// The ThingID of the authn service that handles the request.
	authnServiceID string
}

// AdminAddAgent client method - Add Agent.
// Create an account for IoT device agents
func (m *AuthnAdminMsgClient) AddClient(clientID string, displayName string, role string, pubKey string) (
	token string, err error) {

	var args = authnapi.AdminAddClientArgs{
		ClientID:    clientID,
		DisplayName: displayName,
		Role:        role,
	}
	thingID := authnapi.DefaultAdminServiceID
	err = m.Rpc(td.OpInvokeAction, thingID, authnapi.AdminActionAddClient, &args, &token)
	return
}

// GetClientProfile client method - Get Client Profile.
// Get the profile information describing a client
func (m *AuthnAdminMsgClient) GetClientProfile(clientID string) (
	profile authnapi.ClientProfile, err error) {

	thingID := authnapi.DefaultAdminServiceID
	err = m.Rpc(td.OpInvokeAction, thingID,
		authnapi.AdminActionGetProfile, &clientID, &profile)
	return
}

// GetProfiles client method - Get Profiles.
// Get a list of all client profiles
func (m *AuthnAdminMsgClient) GetProfiles() (clientProfiles []authnapi.ClientProfile, err error) {

	thingID := authnapi.DefaultAdminServiceID
	err = m.Rpc(td.OpInvokeAction, thingID,
		authnapi.AdminActionGetProfiles, nil, &clientProfiles)
	return
}

// RemoveClient client method - Remove Client.
// Remove a client account
func (m *AuthnAdminMsgClient) RemoveClient(clientID string) (err error) {

	thingID := authnapi.DefaultAdminServiceID
	err = m.Rpc(td.OpInvokeAction, thingID,
		authnapi.AdminActionRemoveClient, &clientID, nil)
	return
}

// SetClientPassword client method - Set Client Password.
// Update the password of a consumer
func (m *AuthnAdminMsgClient) SetClientPassword(userName string, password string) (err error) {

	var args = authnapi.AdminSetPasswordArgs{
		UserName: userName, Password: password}

	thingID := authnapi.DefaultAdminServiceID
	err = m.Rpc(td.OpInvokeAction, thingID,
		authnapi.AdminActionSetPassword, &args, nil)
	return
}

// UpdateClientProfile client method - Update Client Profile.
// Update the details of a client
func (m *AuthnAdminMsgClient) UpdateClientProfile(clientProfile authnapi.ClientProfile) (err error) {

	thingID := authnapi.DefaultAdminServiceID
	err = m.Rpc(td.OpInvokeAction,
		authnapi.AuthnAdminServiceID, thingID, &clientProfile, nil)
	return
}

// Create a new instance of the authentication administration messaging client
// sink is the request handler this will link to. nil to ignore.
func NewAuthnAdminClient(sink modules.IHiveModule) *AuthnAdminMsgClient {
	m := &AuthnAdminMsgClient{
		HiveModuleBase: modules.NewHiveModuleBase("", 0),
	}
	if sink != nil {
		m.SetRequestSink(sink)
		sink.SetNotificationSink(m)
	}
	return m
}
