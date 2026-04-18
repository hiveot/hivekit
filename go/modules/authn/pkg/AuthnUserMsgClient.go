// Package authnclient with consumer facing methods.
package authnpkg

import (
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/clients"
)

// AuthnUserMsgClient is a client for authentication operations using RRN messages
// Intended to be linked to a transport client module for message delivery.
type AuthnUserMsgClient struct {
	modules.HiveModuleBase
	// the authn module providing the actual functionality
	co *clients.Consumer
	// The ThingID of the server to connect to
	authServiceID string
}

// UserGetProfile client method - Get Client Profile.
func (cl *AuthnUserMsgClient) GetProfile() (resp authn.ClientProfile, err error) {
	err = cl.co.Rpc(td.OpInvokeAction,
		authn.AuthnUserServiceID,
		authn.UserActionGetProfile, nil, &resp)
	return
}

// // Login with password
// func (cl *AuthnUserMsgClient) Login(clientID string, password string) (token string, err error) {
// 	var args = authn.UserLoginArgs{UserName: clientID, Password: password}
// 	err = cl.cc.Rpc("invokeaction", authn.AuthnUserServiceID, UserLoginAction, &args, &token)
// 	return
// }

// Logout client method - Logout.
// Logout from all devices
func (cl *AuthnUserMsgClient) Logout() (err error) {

	err = cl.co.Rpc(td.OpInvokeAction,
		authn.AuthnUserServiceID,
		authn.UserActionLogout, nil, nil)
	return
}

// UserRefreshToken client method - Request a new auth token for the current client.
func (cl *AuthnUserMsgClient) RefreshToken(hc *clients.Consumer, oldToken string) (newToken string, err error) {

	err = cl.co.Rpc(td.OpInvokeAction,
		authn.AuthnUserServiceID,
		authn.UserActionRefreshToken, &oldToken, &newToken)
	return
}

// UserUpdatePassword client method - Update Password.
// Request changing the password of the current client
func (cl *AuthnUserMsgClient) UpdateProfile(hc *clients.Consumer, password string) (err error) {
	err = cl.co.Rpc(td.OpInvokeAction,
		authn.AuthnUserServiceID,
		authn.UserActionSetPassword, &password, nil)
	return
}

// Create a new instance of the authn messaging consumer client
func NewAuthnUserMsgClient(co *clients.Consumer) *AuthnUserMsgClient {
	cl := &AuthnUserMsgClient{
		co: co,
	}
	return cl
}
