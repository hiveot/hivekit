package authn_test

import (
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/authn"
	authnclient "github.com/hiveot/hivekit/go/modules/authn/client"
	"github.com/hiveot/hivekit/go/modules/transports/clients"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: RefreshToken is only possible when using JWT.
func TestLoginRefresh(t *testing.T) {
	var user1ID = "user1ID"
	var tu1Pass = "tu1Pass"

	svc, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()
	authenticator := svc.GetAuthenticator()

	// add user to test with
	err := svc.AddClient(user1ID, testClientID1, authn.ClientRoleViewer, "")
	require.NoError(t, err)

	err = svc.SetPassword(user1ID, tu1Pass)
	require.NoError(t, err)

	token1, validUntil, err := svc.Login(user1ID, tu1Pass)
	require.NoError(t, err)
	require.Greater(t, validUntil, time.Now())

	cid2, role2, validUntil2, err := authenticator.ValidateToken(token1)
	require.NoError(t, err)
	assert.Equal(t, user1ID, cid2)
	assert.Equal(t, authn.ClientRoleViewer, role2)
	require.Equal(t, validUntil2, validUntil)

	// RefreshToken the token
	token3, validUntil3, err := svc.RefreshToken(user1ID, token1)
	require.NoError(t, err)
	require.NotEmpty(t, token3)
	require.Greater(t, validUntil3, validUntil2)

	// ValidateToken the new token
	cid4, role4, validUntil4, err := authenticator.ValidateToken(token3)
	assert.Equal(t, user1ID, cid4)
	assert.Equal(t, authn.ClientRoleViewer, role4)
	assert.Equal(t, validUntil3, validUntil4)
	require.NoError(t, err)
}

func TestBadRefresh(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	srv, cancelFn := startTestAuthnModule(defaultHash)
	defer cancelFn()

	co1, cc1, token1 := NewTestConsumer(testClientID1, srv.GetAuthenticator())
	_ = co1
	_ = token1
	defer cc1.Close()

	// set the token
	t.Log("Expecting SetBearerToken('bad-token') to fail")
	err := cc1.ConnectWithToken(testClientID1, "bad-token")
	require.Error(t, err)

	// reconnect with a valid token and connect with a bad client-id
	err = cc1.ConnectWithToken(testClientID1, token1)
	assert.NoError(t, err)

	serverURL := srv.GetConnectURL()
	authCl := authnclient.NewAuthnHttpClient(serverURL, testCerts.CaCert)
	authCl.ConnectWithToken(testClientID1, token1)
	validToken, err := authCl.RefreshToken(token1)
	//validToken, err := co1.RefreshToken(token1)
	assert.NoError(t, err)
	assert.NotEmpty(t, validToken)
	cc1.Close()
}

func TestLogout(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	srv, cancelFn := startTestAuthnModule(defaultHash)
	defer cancelFn()

	// check if this test still works with a valid login
	co1, cc1, token1 := NewTestConsumer(testClientID1, srv.GetAuthenticator())
	_ = cc1
	_ = co1
	defer co1.Stop()
	assert.NotEmpty(t, token1)

	// logout
	serverURL := srv.GetConnectURL()
	authnClient := authnclient.NewAuthnHttpClient(serverURL, testCerts.CaCert)
	authnClient.ConnectWithToken(testClientID1, token1)
	err := authnClient.Logout(token1)
	assert.NoError(t, err)

	//authenticator.Logout(cc1, "")
	//err := co1.Logout()
	t.Log(">>> Logged out, an unauthorized error is expected next.")

	// This causes Refresh to fail
	token2, err := authnClient.RefreshToken(token1)
	//token2, err := co1.RefreshToken(token1)
	assert.Error(t, err)
	assert.Empty(t, token2)
}
func TestUpdatePassword(t *testing.T) {

	var user1ID = "user1ID"
	var tu1Name = "test user 1"

	srv, cancelFn := startTestAuthnModule(defaultHash)
	defer cancelFn()

	// add user to test with
	co := clients.NewConsumer("test", co, 0)
	tp := direct.NewDirectTransport(user1ID, co, srv)

	authCl := authnclient.NewAuthnUserMsgClient(co)
	err := svc.AdminSvc.AddConsumer(user1ID, authn.AdminAddConsumerArgs{user1ID, tu1Name, "oldpass"})
	require.NoError(t, err)

	// login should succeed
	_, err = svc.UserSvc.Login(user1ID, authn.UserLoginArgs{user1ID, "oldpass"})
	require.NoError(t, err)

	// change password
	err = svc.UserSvc.UpdatePassword(user1ID, "newpass")
	require.NoError(t, err)

	// login with old password should now fail
	//t.Log("an error is expected logging in with the old password")
	_, err = svc.UserSvc.Login(user1ID, authn.UserLoginArgs{user1ID, "oldpass"})
	require.Error(t, err)

	// re-login with new password
	_, err = svc.UserSvc.Login(user1ID, authn.UserLoginArgs{user1ID, "newpass"})
	require.NoError(t, err)
}

func TestUpdatePasswordFail(t *testing.T) {
	var user1ID = "user1ID"
	srv, cancelFn := startTestAuthnModule(defaultHash)
	defer cancelFn()

	err := svc.UserSvc.UpdatePassword(user1ID, "newpass")
	assert.Error(t, err)
}

func TestUpdateName(t *testing.T) {

	var user1ID = "user1ID"
	var tu1Name = "test user 1"
	var tu2Name = "test user 1"

	srv, cancelFn := startTestAuthnModule(defaultHash)
	defer cancelFn()

	// add user to test with
	err := svc.AdminSvc.AddConsumer(user1ID, authn.AdminAddConsumerArgs{user1ID, tu1Name, "oldpass"})
	require.NoError(t, err)

	profile, err := svc.UserSvc.GetProfile(user1ID)
	require.NoError(t, err)
	assert.Equal(t, tu1Name, profile.DisplayName)

	err = svc.UserSvc.UpdateName(user1ID, tu2Name)
	require.NoError(t, err)
	profile2, err := svc.UserSvc.GetProfile(user1ID)
	require.NoError(t, err)

	assert.Equal(t, tu2Name, profile2.DisplayName)
}

func TestClientUpdatePubKey(t *testing.T) {
	var user1ID = "user1ID"

	srv, cancelFn := startTestAuthnModule(defaultHash)
	defer cancelFn()

	// add user to test with. don't set the public key yet
	err := svc.AdminSvc.AddClient("",
		authn.AdminAddConsumerArgs{user1ID, user1ID, "user1"})
	profile, err := svc.AdminSvc.GetClientProfile("", user1ID)
	require.NoError(t, err)
	assert.Equal(t, user1ID, profile.ClientID)
	assert.Equal(t, user1ID, profile.DisplayName)
	assert.NotEmpty(t, profile.Updated)

	// update the public key
	privKey, pubKey := utils.NewKey(utils.KeyTypeECDSA)
	_ = privKey
	profile2, err := svc.UserSvc.GetProfile(user1ID)
	assert.Equal(t, user1ID, profile2.ClientID)
	require.NoError(t, err)
	err = svc.UserSvc.UpdatePubKey(user1ID, pubKey)
	assert.NoError(t, err)

	// check result
	profile3, err := svc.UserSvc.GetProfile(user1ID)
	require.NoError(t, err)
	assert.Equal(t, user1ID, profile3.ClientID)
	assert.Equal(t, kp.ExportPublic(), profile3.PubKey)
}
