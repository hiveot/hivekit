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

	token1, validUntil, err := authenticator.Login(user1ID, tu1Pass)
	require.NoError(t, err)
	require.Greater(t, validUntil, time.Now())

	cid2, role2, validUntil2, err := authenticator.ValidateToken(token1)
	require.NoError(t, err)
	assert.Equal(t, user1ID, cid2)
	assert.Equal(t, string(authn.ClientRoleViewer), role2)
	require.Equal(t, validUntil2, validUntil)

	// RefreshToken the token after a short delay
	token3, validUntil3, err := authenticator.RefreshToken(user1ID, token1)
	require.NoError(t, err)
	require.NotEmpty(t, token3)

	// ValidateToken the new token
	cid4, role4, validUntil4, err := authenticator.ValidateToken(token3)
	assert.Equal(t, user1ID, cid4)
	assert.Equal(t, string(authn.ClientRoleViewer), role4)
	assert.Equal(t, validUntil3, validUntil4)
	require.NoError(t, err)
}

func TestBadRefresh(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	srv, cancelFn := startTestAuthnModule(defaultHash)
	defer cancelFn()
	serverURL := srv.GetConnectURL()

	co1, cc1, token1 := NewTestConsumer(srv, serverURL, testClientID1)
	_ = co1
	_ = token1
	defer cc1.Close()

	// set the token
	t.Log("Expecting SetBearerToken('bad-token') to fail")
	err := cc1.ConnectWithToken(testClientID1, "bad-token", nil)
	// TODO: http-basic doesn't check tokens until a request is sent to the protected endpoint
	if err == nil {
		err = co1.Ping() // hiveot support ping, although it doesnt require authn
	}
	assert.Error(t, err)

	// reconnect with a valid token and connect with a bad client-id
	err = cc1.ConnectWithToken(testClientID1, token1, nil)
	assert.NoError(t, err)

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
	serverURL := srv.GetConnectURL()

	// check if this test still works with a valid login
	co1, cc1, token1 := NewTestConsumer(srv, serverURL, testClientID1)
	_ = cc1
	_ = co1
	defer co1.Stop()
	assert.NotEmpty(t, token1)

	// logout
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
	authenticator := srv.GetAuthenticator()

	tp := direct.NewDirectTransport(user1ID, srv)

	// add user to test with
	co := clients.NewConsumer("test", 0)
	authCl := authnclient.NewAuthnUserMsgClient(co)
	authCl.SetRequestSink(tp.HandleRequest)

	err := srv.AddClient(user1ID, tu1Name, authn.ClientRoleViewer, "oldpass")
	srv.SetPassword(user1ID, "oldpass")
	require.NoError(t, err)

	// login should succeed
	_, _, err = authenticator.Login(user1ID, "oldpass")
	require.NoError(t, err)

	// change password
	err = srv.SetPassword(user1ID, "newpass")
	require.NoError(t, err)

	// login with old password should now fail
	//t.Log("an error is expected logging in with the old password")
	_, _, err = authenticator.Login(user1ID, "oldpass")
	require.Error(t, err)

	// re-login with new password
	_, _, err = authenticator.Login(user1ID, "newpass")
	require.NoError(t, err)
}

func TestUpdatePasswordFail(t *testing.T) {
	var user1ID = "user1ID"
	srv, cancelFn := startTestAuthnModule(defaultHash)
	defer cancelFn()

	err := srv.SetPassword(user1ID, "newpass")
	assert.Error(t, err)
}

func TestUpdateName(t *testing.T) {

	var user1ID = "user1ID"
	var tu1Name = "test user 1"
	var tu2Name = "test user 1"

	srv, cancelFn := startTestAuthnModule(defaultHash)
	defer cancelFn()

	// add user to test with
	err := srv.AddClient(user1ID, tu1Name, authn.ClientRoleViewer, "")
	srv.SetPassword(user1ID, "oldpass")
	require.NoError(t, err)

	profile, err := srv.GetProfile(user1ID)
	require.NoError(t, err)
	assert.Equal(t, tu1Name, profile.DisplayName)

	profile.DisplayName = tu2Name
	err = srv.UpdateProfile(user1ID, profile)
	require.NoError(t, err)
	profile2, err := srv.GetProfile(user1ID)
	require.NoError(t, err)

	assert.Equal(t, tu2Name, profile2.DisplayName)
}

func TestClientUpdatePubKey(t *testing.T) {
	var user1ID = "user1ID"

	srv, cancelFn := startTestAuthnModule(defaultHash)
	defer cancelFn()

	// add user to test with. don't set the public key yet
	err := srv.AddClient(user1ID, user1ID, authn.ClientRoleViewer, "")
	srv.SetPassword(user1ID, "user1")
	profile, err := srv.GetProfile(user1ID)
	require.NoError(t, err)
	assert.Equal(t, user1ID, profile.ClientID)
	assert.Equal(t, user1ID, profile.DisplayName)
	assert.NotEmpty(t, profile.TimeUpdated)

	// update the public key
	privKey, pubKey := utils.NewKey(utils.KeyTypeECDSA)
	pubKeyPem := utils.PublicKeyToPem(pubKey)
	_ = privKey
	profile2, err := srv.GetProfile(user1ID)
	assert.Equal(t, user1ID, profile2.ClientID)
	require.NoError(t, err)
	profile2.PubKeyPem = pubKeyPem
	err = srv.UpdateProfile(user1ID, profile2)
	assert.NoError(t, err)

	// check result
	profile3, err := srv.GetProfile(user1ID)
	require.NoError(t, err)
	assert.Equal(t, user1ID, profile3.ClientID)
	assert.Equal(t, pubKeyPem, profile3.PubKeyPem)
}
