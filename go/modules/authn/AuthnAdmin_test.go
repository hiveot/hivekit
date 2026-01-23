package authn_test

import (
	"testing"

	"github.com/hiveot/hivekit/go/modules/authn"
	authnserver "github.com/hiveot/hivekit/go/modules/authn/server"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: this uses default settings from Authn_test.go

// Create manage users
func TestAddRemoveClientsSuccess(t *testing.T) {
	deviceID := "device1"
	devicePrivKey, devicePubKey := utils.NewKey(utils.KeyTypeECDSA)
	serviceID := "service1"
	servicePrivKey, servicePubKey := utils.NewKey(utils.KeyTypeECDSA)

	m, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()
	//hc := embedded.NewEmbeddedClient(serviceID, adminHandler)

	//err := svc.AdminSvc.AddConsumer(serviceID,
	//         authn.AdminAddConsumerArgs{ "user1", "user 1", "pass1")
	err := m.AddClient(serviceID,
		authnserver.AdminAddClientArgs{
			ClientID:    "user1",
			DisplayName: "user 1",
			Role:        authn.ClientRoleViewer,
			Password:    "pass1",
		})
	assert.NoError(t, err)
	// duplicate should update
	err = m.AdminSvc.AddConsumer(serviceID,
		authn.AdminAddConsumerArgs{ClientID: "user1", DisplayName: "user 1 updated", Password: "pass1"})
	assert.NoError(t, err)

	err = m.AdminSvc.AddConsumer(serviceID,
		authn.AdminAddConsumerArgs{ClientID: "user2", DisplayName: "user 2", Password: "pass2"})
	assert.NoError(t, err)
	err = m.AdminSvc.AddConsumer(serviceID,
		authn.AdminAddConsumerArgs{ClientID: "user3", DisplayName: "user 3", Password: "pass2"})
	assert.NoError(t, err)
	err = m.AdminSvc.AddConsumer(serviceID,
		authn.AdminAddConsumerArgs{ClientID: "user4", DisplayName: "user 4", Password: "pass2"})
	assert.NoError(t, err)

	_, err = m.AdminSvc.AddAgent(serviceID,
		authn.AdminAddAgentArgs{ClientID: deviceID, DisplayName: "agent 1", PubKey: deviceKeyPub})
	assert.NoError(t, err)

	_, err = m.AdminSvc.AddService(serviceID,
		authn.AdminAddServiceArgs{ClientID: serviceID, DisplayName: "service 1", PubKey: serviceKeyPub})
	assert.NoError(t, err)

	// update the server. users can connect and have unlimited access
	profiles, err := m.AdminSvc.GetProfiles(serviceID)
	require.NoError(t, err)
	assert.Equal(t, 6+3, len(profiles))

	err = m.AdminSvc.RemoveClient(serviceID, "user1")
	assert.NoError(t, err)
	err = m.AdminSvc.RemoveClient(serviceID, "user1") // remove is idempotent
	assert.NoError(t, err)
	err = m.AdminSvc.RemoveClient(serviceID, "user2")
	assert.NoError(t, err)
	err = m.AdminSvc.RemoveClient(serviceID, deviceID)
	assert.NoError(t, err)
	err = m.AdminSvc.RemoveClient(serviceID, serviceID)
	assert.NoError(t, err)

	profiles, err = m.AdminSvc.GetProfiles(serviceID)

	require.NoError(t, err)
	assert.Equal(t, 2+3, len(profiles))

	clEntries := m.AdminSvc.GetEntries()
	assert.Equal(t, 2+3, len(clEntries))

	err = m.AdminSvc.AddConsumer(serviceID,
		authn.AdminAddConsumerArgs{"user1", "user 1", "pass1"})
	assert.NoError(t, err)
}

// Create manage users
func TestAddRemoveClientsFail(t *testing.T) {
	const adminID = "administrator-1"
	m, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()

	// missing clientID should fail
	_, err := m.AdminSvc.AddService(adminID, authn.AdminAddServiceArgs{"", "user 1", ""})
	assert.Error(t, err)

	// a bad key is not an error
	err = m.AdminSvc.AddConsumer(adminID, authn.AdminAddConsumerArgs{"user2", "user 2", "badkey"})
	assert.NoError(t, err)
}

func TestUpdateClientPassword(t *testing.T) {
	var tu1ID = "tu1ID"
	var tuPass1 = "tuPass1"
	var tuPass2 = "tuPass2"
	const adminID = "administrator-1"

	m, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()
	err := m.AdminSvc.AddConsumer(
		adminID, authn.AdminAddConsumerArgs{tu1ID, "user 1", tuPass1})
	require.NoError(t, err)

	err = m.SessionAuth.ValidatePassword(tu1ID, tuPass1)
	require.NoError(t, err)

	err = m.AdminSvc.SetClientPassword(
		adminID, authn.AdminSetClientPasswordArgs{tu1ID, tuPass2})
	require.NoError(t, err)

	err = m.SessionAuth.ValidatePassword(tu1ID, tuPass1)
	require.Error(t, err)

	err = m.SessionAuth.ValidatePassword(tu1ID, tuPass2)
	require.NoError(t, err)
}

func TestUpdatePubKey(t *testing.T) {
	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"
	const adminID = "administrator-1"

	m, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()

	// add user to test with. don't set the public key yet
	err := m.AdminSvc.AddConsumer(adminID, authn.AdminAddConsumerArgs{tu1ID, "user 2", tu1Pass})
	require.NoError(t, err)
	//
	token := m.SessionAuth.CreateSessionToken(tu1ID, "", 0)
	require.NotEmpty(t, token)

	// update the public key
	kp := utils.NewKey(utils.KeyTypeECDSA)
	profile, err := m.AdminSvc.GetClientProfile(adminID, tu1ID)
	require.NoError(t, err)
	profile.PubKey = kp.ExportPublic()
	err = m.AdminSvc.UpdateClientProfile(adminID, profile)
	assert.NoError(t, err)

	// check result
	profile2, err := m.AdminSvc.GetClientProfile(adminID, tu1ID)
	require.NoError(t, err)
	assert.Equal(t, kp.ExportPublic(), profile2.PubKey)
}

func TestNewAgentToken(t *testing.T) {
	var tu1ID = "ag1ID"
	var tu1Name = "agent 1"

	const adminID = "administrator-1"
	m, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()

	// add agent to test with and connect
	_, err := m.AdminSvc.AddAgent(adminID, authn.AdminAddAgentArgs{tu1ID, tu1Name, ""})
	require.NoError(t, err)

	// get a new token
	token, err := m.AdminSvc.NewAgentToken(adminID, tu1ID)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// login with new token
	clientID, _, err := m.SessionAuth.ValidateToken(token)
	require.NoError(t, err)
	require.Equal(t, tu1ID, clientID)

}

func TestUpdateProfile(t *testing.T) {
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	const adminID = "administrator-1"
	m, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()

	// add user to test with and connect
	err := m.AdminSvc.AddConsumer(adminID, authn.AdminAddConsumerArgs{tu1ID, tu1Name, "pass0"})
	require.NoError(t, err)
	//tu1Key, _ := testServer.MsgServer.CreateKP()

	// update display name
	const newDisplayName = "new display name"
	profile, err := m.AdminSvc.GetClientProfile(adminID, tu1ID)
	require.NoError(t, err)
	profile.DisplayName = newDisplayName
	err = m.AdminSvc.UpdateClientProfile(adminID, profile)
	assert.NoError(t, err)

	// verify
	profile2, err := m.AdminSvc.GetClientProfile(adminID, tu1ID)

	require.NoError(t, err)
	assert.Equal(t, newDisplayName, profile2.DisplayName)
}

func TestUpdateProfileFail(t *testing.T) {
	const adminID = "administrator-1"

	m, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()

	err := m.AdminSvc.UpdateClientProfile(adminID, authn.ClientProfile{ClientID: "badclient"})
	assert.Error(t, err)
}
