// Package authnclient with consumer facing methods.
package authnclient

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/authn"
)

// AuthnUserClient is a client for authentication operations using RRN messages.
// This should be linked to a transport client cell for message delivery.
type AuthnUserClient struct {
	*cells.HiveCellBase
	// The ThingID of the authentication service that handles the request.
	authnServiceID string
}

// UserGetProfile client method - Get Client Profile.
func (m *AuthnUserClient) GetProfile() (resp authn.ClientProfile, err error) {
	err = m.Rpc(td.OpInvokeAction,
		authn.AuthnUserServiceID, authn.UserActionGetProfile, nil, &resp)
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
func (m *AuthnUserClient) Logout() (err error) {

	err = m.Rpc(td.OpInvokeAction,
		authn.AuthnUserServiceID, authn.UserActionLogout, nil, nil)
	return
}

// UserRefreshToken client method - Request a new auth token for the current client.
func (m *AuthnUserClient) RefreshToken(oldToken string) (newToken string, err error) {

	err = m.Rpc(td.OpInvokeAction,
		authn.AuthnUserServiceID, authn.UserActionRefreshToken, &oldToken, &newToken)
	return
}

// UserUpdatePassword client method - Update Password.
// Request changing the password of the current client
func (m *AuthnUserClient) UpdateProfile(password string) (err error) {
	err = m.Rpc(td.OpInvokeAction,
		authn.AuthnUserServiceID, authn.UserActionSetPassword, &password, nil)
	return
}

// Create a new instance of the authn user client
//
// sink is the chain containing the user's transport client
func NewAuthnUserClient(sink api.IHiveCell) *AuthnUserClient {
	cl := &AuthnUserClient{
		HiveCellBase: cells.NewHiveCellBase("", 0),
	}
	if sink != nil {
		cl.SetRequestSink(sink)
		sink.SetNotificationSink(cl)
	}
	return cl
}
