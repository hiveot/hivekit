package authnclient

import (
	"github.com/hiveot/hivekit/go/lib/consumer"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/authn/server"
	"github.com/hiveot/hivekit/go/wot"
)

// AdminAddAgent client method - Add Agent.
// Create an account for IoT device agents
func AdminAddClient(hc *consumer.Consumer, clientID string, displayName string, pubKey string, role authn.ClientRole) (
	token string, err error) {

	var args = server.AdminAddClientArgs{
		ClientID:    clientID,
		DisplayName: displayName,
		PubKey:      pubKey,
		Role:        role}
	err = hc.Rpc(wot.OpInvokeAction,
		authn.AdminServiceID, server.AdminActionAddClient, &args, &token)
	return
}

// AdminGetClientProfile client method - Get Client Profile.
// Get the profile information describing a client
func AdminGetClientProfile(hc *consumer.Consumer, clientID string) (
	profile authn.ClientProfile, err error) {

	err = hc.Rpc(wot.OpInvokeAction,
		authn.AdminServiceID,
		server.AdminActionGetProfile, &clientID, &profile)
	return
}

// AdminGetProfiles client method - Get Profiles.
// Get a list of all client profiles
func AdminGetProfiles(hc *consumer.Consumer) (clientProfiles []authn.ClientProfile, err error) {

	err = hc.Rpc(wot.OpInvokeAction,
		server.AuthnAdminServiceID,
		server.AdminActionGetProfiles, nil, &clientProfiles)
	return
}

// AdminRemoveClient client method - Remove Client.
// Remove a client account
func AdminRemoveClient(hc *consumer.Consumer, clientID string) (err error) {

	err = hc.Rpc(wot.OpInvokeAction,
		server.AuthnAdminServiceID,
		server.AdminActionRemoveClient, &clientID, nil)
	return
}

// AdminSetClientPassword client method - Set Client Password.
// Update the password of a consumer
func AdminSetClientPassword(hc *consumer.Consumer, userName string, password string) (err error) {
	var args = server.AdminSetPasswordArgs{
		UserName: userName, Password: password}
	err = hc.Rpc(wot.OpInvokeAction,
		server.AuthnAdminServiceID,
		server.AdminActionSetPassword, &args, nil)
	return
}

// AdminUpdateClientProfile client method - Update Client Profile.
// Update the details of a client
func AdminUpdateClientProfile(hc *consumer.Consumer, clientProfile authn.ClientProfile) (err error) {

	err = hc.Rpc(wot.OpInvokeAction,
		server.AuthnAdminServiceID,
		server.AdminActionUpdateProfile, &clientProfile, nil)
	return
}
