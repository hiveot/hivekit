// Package authnclient with administration facing methods
package authnclient

import (
	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/wot"
)

// AdminAddAgent client method - Add Agent.
// Create an account for IoT device agents
func AdminAddClient(hc *clients.Consumer, clientID string, displayName string, role string, pubKey string) (
	token string, err error) {

	var args = authnapi.AdminAddClientArgs{
		ClientID:    clientID,
		DisplayName: displayName,
		Role:        role,
	}
	thingID := authnapi.DefaultAdminServiceID
	err = hc.Rpc(wot.OpInvokeAction, thingID, authnapi.AdminActionAddClient, &args, &token)
	return
}

// AdminGetClientProfile client method - Get Client Profile.
// Get the profile information describing a client
func AdminGetClientProfile(hc *clients.Consumer, clientID string) (
	profile authnapi.ClientProfile, err error) {

	thingID := authnapi.DefaultAdminServiceID
	err = hc.Rpc(wot.OpInvokeAction, thingID,
		authnapi.AdminActionGetProfile, &clientID, &profile)
	return
}

// AdminGetProfiles client method - Get Profiles.
// Get a list of all client profiles
func AdminGetProfiles(hc *clients.Consumer) (clientProfiles []authnapi.ClientProfile, err error) {

	thingID := authnapi.DefaultAdminServiceID
	err = hc.Rpc(wot.OpInvokeAction, thingID,
		authnapi.AdminActionGetProfiles, nil, &clientProfiles)
	return
}

// AdminRemoveClient client method - Remove Client.
// Remove a client account
func AdminRemoveClient(hc *clients.Consumer, clientID string) (err error) {

	thingID := authnapi.DefaultAdminServiceID
	err = hc.Rpc(wot.OpInvokeAction, thingID,
		authnapi.AdminActionRemoveClient, &clientID, nil)
	return
}

// AdminSetClientPassword client method - Set Client Password.
// Update the password of a consumer
func AdminSetClientPassword(hc *clients.Consumer, userName string, password string) (err error) {
	var args = authnapi.AdminSetPasswordArgs{
		UserName: userName, Password: password}

	thingID := authnapi.DefaultAdminServiceID
	err = hc.Rpc(wot.OpInvokeAction, thingID,
		authnapi.AdminActionSetPassword, &args, nil)
	return
}

// AdminUpdateClientProfile client method - Update Client Profile.
// Update the details of a client
func AdminUpdateClientProfile(hc *clients.Consumer, clientProfile authnapi.ClientProfile) (err error) {

	thingID := authnapi.DefaultAdminServiceID
	err = hc.Rpc(wot.OpInvokeAction,
		authnapi.AuthnAdminServiceID, thingID, &clientProfile, nil)
	return
}
