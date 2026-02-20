package authn_test

import (
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: this uses default settings from Authn_test.go

// Test the admin messaging interface
// Manage users
func TestAddRemoveClientsSuccess(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	deviceID := "device1"
	devicePrivKey, devicePubKey := utils.NewKey(utils.KeyTypeECDSA)
	_ = devicePrivKey
	serviceID := "service1"
	servicePrivKey, servicePubKey := utils.NewKey(utils.KeyTypeECDSA)
	_ = servicePrivKey

	m, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()
	//hc := embedded.NewEmbeddedClient(serviceID, adminHandler)

	//err := svc.AdminSvc.AddConsumer(serviceID,
	//         authn.AdminAddConsumerArgs{ "user1", "user 1", "pass1")
	servicePubKeyPem := utils.PublicKeyToPem(servicePubKey)
	err := m.AddClient("user1", "User 1", authn.ClientRoleViewer, servicePubKeyPem)
	require.NoError(t, err)
	err2 := m.SetPassword("user1", "pass1")
	require.NoError(t, err2)

	// duplicate should fail
	err = m.AddClient("user1", "user 1 updated", authn.ClientRoleViewer, "")
	require.Error(t, err)

	err = m.AddClient("user2", "user 2", authn.ClientRoleViewer, "")
	assert.NoError(t, err)
	err = m.AddClient("user3", "user 3", authn.ClientRoleViewer, "")
	assert.NoError(t, err)
	err = m.AddClient("user4", "user 4", authn.ClientRoleViewer, "")
	assert.NoError(t, err)

	deviceKeyPubPem := utils.PublicKeyToPem(devicePubKey)
	err = m.AddClient(deviceID, "agent 1", authn.ClientRoleAgent, deviceKeyPubPem)
	assert.NoError(t, err)

	serviceKeyPubPem := utils.PublicKeyToPem(servicePubKey)
	err = m.AddClient(serviceID, "service 1", authn.ClientRoleService, serviceKeyPubPem)
	assert.NoError(t, err)

	// there should be 6 clients
	profiles, err := m.GetProfiles()
	require.NoError(t, err)
	assert.Equal(t, 6, len(profiles))

	err = m.RemoveClient("user1")
	assert.NoError(t, err)
	err = m.RemoveClient("user1") // remove is idempotent
	assert.NoError(t, err)
	err = m.RemoveClient("user2")
	assert.NoError(t, err)
	err = m.RemoveClient(deviceID)
	assert.NoError(t, err)
	err = m.RemoveClient(serviceID)
	assert.NoError(t, err)

	profiles, err = m.GetProfiles()
	// two accounts remaining (user 3 and 4)
	require.NoError(t, err)
	assert.Equal(t, 2, len(profiles))

	err = m.AddClient("user1", "user 1", authn.ClientRoleViewer, "")
	m.SetPassword("user1", "pass1")
	assert.NoError(t, err)
}

// Create manage users
func TestAddRemoveClientsFail(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const adminID = "administrator-1"
	m, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()

	// missing clientID should fail
	err := m.AddClient("", "user 1", authn.ClientRoleService, "")
	assert.Error(t, err)

	// a bad key is not an error
	err = m.AddClient("user2", "user 2", authn.ClientRoleViewer, "")
	assert.NoError(t, err)
}

func TestUpdateClientPassword(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var tu1ID = "tu1ID"
	var tuPass1 = "tuPass1"
	var tuPass2 = "tuPass2"
	const adminID = "administrator-1"

	m, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()
	err := m.AddClient(tu1ID, "user tu1", authn.ClientRoleViewer, "")
	require.NoError(t, err)
	err = m.SetPassword(tu1ID, tuPass1)
	require.NoError(t, err)

	err = m.ValidatePassword(tu1ID, tuPass1)
	require.NoError(t, err)

	err = m.SetPassword(tu1ID, tuPass2)
	require.NoError(t, err)

	err = m.ValidatePassword(tu1ID, tuPass1)
	require.Error(t, err)

	err = m.ValidatePassword(tu1ID, tuPass2)
	require.NoError(t, err)
}

func TestUpdatePubKey(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"

	m, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()

	// add user to test with. don't set the public key yet
	err := m.AddClient(tu1ID, "user tu1", authn.ClientRoleViewer, "")
	m.SetPassword(tu1ID, tu1Pass)
	require.NoError(t, err)
	//
	token, validUntil, err := m.CreateSessionToken(tu1ID, time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, validUntil)

	// update the public key
	privKey, pubKey := utils.NewKey(utils.KeyTypeECDSA)
	require.NotEmpty(t, privKey)
	profile, err := m.GetProfile(tu1ID)
	require.NoError(t, err)
	profile.PubKeyPem = utils.PublicKeyToPem(pubKey)
	err = m.UpdateProfile(tu1ID, profile)
	assert.NoError(t, err)

	// check result
	profile2, err := m.GetProfile(tu1ID)
	require.NoError(t, err)
	assert.Equal(t, profile.PubKeyPem, profile2.PubKeyPem)
}

func TestNewAgentToken(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var tu1ID = "ag1ID"
	var tu1Name = "agent 1"

	const adminID = "administrator-1"
	m, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()

	// add agent to test with and connect
	err := m.AddClient(tu1ID, tu1Name, authn.ClientRoleAgent, "")
	require.NoError(t, err)

	// get a new token
	token, _, err := m.CreateSessionToken(tu1ID, time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// login with new token
	clientID, role, _, err := m.ValidateToken(token)
	require.NoError(t, err)
	require.Equal(t, tu1ID, clientID)
	require.Equal(t, string(authn.ClientRoleAgent), role)

}

func TestUpdateProfile(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	// const adminID = "administrator-1"
	m, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()

	// add user to test with and connect
	err := m.AddClient(tu1ID, tu1Name, authn.ClientRoleViewer, "")
	require.NoError(t, err)
	//tu1Key, _ := testServer.MsgServer.CreateKP()

	// client can update display name
	const newDisplayName = "new display name"
	profile, err := m.GetProfile(tu1ID)
	require.NoError(t, err)
	profile.DisplayName = newDisplayName
	err = m.UpdateProfile(tu1ID, profile)
	assert.NoError(t, err)

	// verify
	profile2, err := m.GetProfile(tu1ID)

	require.NoError(t, err)
	assert.Equal(t, newDisplayName, profile2.DisplayName)
}

func TestUpdateProfileFail(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const adminID = "administrator-1"
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	m, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()
	// add user to test with and connect
	err := m.AddClient(tu1ID, tu1Name, authn.ClientRoleViewer, "")
	require.NoError(t, err)

	// this fails as badclient doesn't exist
	err = m.UpdateProfile(adminID, authn.ClientProfile{ClientID: "badclient"})
	assert.Error(t, err)
}
