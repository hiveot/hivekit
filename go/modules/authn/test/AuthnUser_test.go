package authn_test

import (
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/authn"
	authnpkg "github.com/hiveot/hivekit/go/modules/authn/pkg"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// test logging in and token refresh using the module directly
func TestLoginRefresh(t *testing.T) {
	var user1ID = "user1ID"
	var tu1Pass = "tu1Pass"

	_, svc, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()

	// add user to test with
	err := svc.AddClient(user1ID, testClientID1, authn.ClientRoleViewer)
	require.NoError(t, err)
	err = svc.SetPassword(user1ID, tu1Pass)
	require.NoError(t, err)

	// first test the login/refresh natively
	sm := svc.GetSessionManager()
	token1, validUntil, err := sm.Login(user1ID, tu1Pass)
	require.NoError(t, err)
	require.Greater(t, validUntil, time.Now())

	cid2, _, validUntil2, err := sm.ValidateToken(token1)
	require.NoError(t, err)
	assert.Equal(t, user1ID, cid2)
	require.Equal(t, validUntil2, validUntil)

	token3, validUntil3, err := sm.RefreshToken(user1ID, token1)
	require.NoError(t, err)
	require.NotEmpty(t, token3)

	// ValidateToken the new token
	cid4, _, validUntil4, err := sm.ValidateToken(token3)
	assert.Equal(t, user1ID, cid4)
	assert.Equal(t, validUntil3, validUntil4)
	require.NoError(t, err)
}

func TestUpdatePassword(t *testing.T) {

	var user1ID = "user1ID"
	var tu1Name = "test user 1"

	httpServer, svc, cancelFn := startTestAuthnModule(defaultHash)
	_ = httpServer
	defer cancelFn()

	tp := testenv.NewTestTransport(user1ID, svc)

	// add user to test with
	authCl := authnpkg.NewAuthnUserMsgClient()
	authCl.SetRequestSink(tp.HandleRequest)

	err := svc.AddClient(user1ID, tu1Name, authn.ClientRoleViewer)
	svc.SetPassword(user1ID, "oldpass")
	require.NoError(t, err)

	// login should succeed
	sm := svc.GetSessionManager()
	_, _, err = sm.Login(user1ID, "oldpass")
	require.NoError(t, err)

	// change password
	err = svc.SetPassword(user1ID, "newpass")
	require.NoError(t, err)

	// login with old password should now fail
	//t.Log("an error is expected logging in with the old password")
	_, _, err = sm.Login(user1ID, "oldpass")
	require.Error(t, err)

	// re-login with new password
	_, _, err = sm.Login(user1ID, "newpass")
	require.NoError(t, err)
}

func TestUpdatePasswordFail(t *testing.T) {
	var user1ID = "user1ID"
	httpServer, m, cancelFn := startTestAuthnModule(defaultHash)
	_ = httpServer
	defer cancelFn()

	err := m.SetPassword(user1ID, "newpass")
	assert.Error(t, err)
}

func TestUpdateName(t *testing.T) {

	var user1ID = "user1ID"
	var tu1Name = "test user 1"
	var tu2Name = "test user 1"

	httpServer, m, cancelFn := startTestAuthnModule(defaultHash)
	_ = httpServer
	defer cancelFn()

	// add user to test with
	err := m.AddClient(user1ID, tu1Name, authn.ClientRoleViewer)
	m.SetPassword(user1ID, "oldpass")
	require.NoError(t, err)

	profile, err := m.GetProfile(user1ID)
	require.NoError(t, err)
	assert.Equal(t, tu1Name, profile.DisplayName)

	profile.DisplayName = tu2Name
	err = m.UpdateProfile(user1ID, profile)
	require.NoError(t, err)
	profile2, err := m.GetProfile(user1ID)
	require.NoError(t, err)

	assert.Equal(t, tu2Name, profile2.DisplayName)
}

func TestClientUpdatePubKey(t *testing.T) {
	var user1ID = "user1ID"

	httpServer, m, cancelFn := startTestAuthnModule(defaultHash)
	_ = httpServer
	defer cancelFn()

	// add user to test with. don't set the public key yet
	err := m.AddClient(user1ID, user1ID, authn.ClientRoleViewer)
	m.SetPassword(user1ID, "user1")
	profile, err := m.GetProfile(user1ID)
	require.NoError(t, err)
	assert.Equal(t, user1ID, profile.ClientID)
	assert.Equal(t, user1ID, profile.DisplayName)
	assert.NotEmpty(t, profile.TimeUpdated)

	// update the public key
	privKey, pubKey := utils.NewKey(utils.KeyTypeECDSA)
	pubKeyPem := utils.PublicKeyToPem(pubKey)
	_ = privKey
	profile2, err := m.GetProfile(user1ID)
	assert.Equal(t, user1ID, profile2.ClientID)
	require.NoError(t, err)
	profile2.PubKeyPem = pubKeyPem
	err = m.UpdateProfile(user1ID, profile2)
	assert.NoError(t, err)

	// check result
	profile3, err := m.GetProfile(user1ID)
	require.NoError(t, err)
	assert.Equal(t, user1ID, profile3.ClientID)
	assert.Equal(t, pubKeyPem, profile3.PubKeyPem)
}
