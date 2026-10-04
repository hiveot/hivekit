package authn_test

import (
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/cells/authn"
	certstest "github.com/hiveot/hivekit/go/cells/certs/test"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var serverPort int = 9445
var testCerts certstest.TestCertBundle
var testClientID1 = "client1"
var rpcTimeout = time.Minute * 3

const TestKeyType = utils.KeyTypeED25519

// TestMain creates a test environment
// Used for all test cases in this package
// func TestMain(m *testing.M) {
// 	utils.SetLogging("info", "")
// 	res := m.Run()
// 	if res == 0 {
// 		_ = os.RemoveAll(testDir)
// 	}
// 	os.Exit(res)
// }

// NewTestConsumer creates a new connected consumer client with the given ID.
// The transport server must be started first.
//
// This uses the clientID as password
// This panics if a client cannot be created
// func NewTestConsumer(m *service.AuthnService, protocolType, serverURL, clientID string) (
// 	*consumer.Consumer, api.ITransportClient, string) {

// 	// ensure the client exists
// 	_ = m.AddClient(clientID, "client 1", authn.ClientRoleViewer)
// 	sm := m.GetSessionManager()
// 	token, validUntil, _ := sm.CreateToken(clientID, time.Minute)
// 	_ = validUntil
// 	co, cc, err := clients.NewConsumerConnection(
// 		appID, protocolType, serverURL, nil, testCerts.CaCert, rpcTimeout)
// 	if err != nil {
// 		panic("Failed creating consumer connection: " + err.Error())
// 	}
// 	cc.SetAuthToken(clientID, token)

// 	return co, cc, token
// }

// NOTE: this uses default settings from Authn_test.go

// Test the admin messaging interface
// Manage users
func TestAddRemoveClientsSuccess(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	deviceID := "device1"
	// devicePrivKey, devicePubKey := utils.NewKey(utils.KeyTypeECDSA)
	// _ = devicePrivKey
	serviceID := "service1"
	// servicePrivKey, servicePubKey := utils.NewKey(utils.KeyTypeECDSA)
	// _ = servicePrivKey

	_, authnSvc, stopFn := startTestAuthnServices(defaultHash)
	adminSvc := authnSvc.GetAdminService()
	defer stopFn()

	// add the client that is going to be authenticated
	err := adminSvc.AddClient("user1", "User 1", authn.ClientRoleViewer)
	require.NoError(t, err)
	err2 := adminSvc.SetPassword("user1", "pass1")
	require.NoError(t, err2)

	// duplicate should fail
	err = adminSvc.AddClient("user1", "user 1 updated", authn.ClientRoleViewer)
	require.Error(t, err)

	err = adminSvc.AddClient("user2", "user 2", authn.ClientRoleViewer)
	assert.NoError(t, err)
	err = adminSvc.AddClient("user3", "user 3", authn.ClientRoleViewer)
	assert.NoError(t, err)
	err = adminSvc.AddClient("user4", "user 4", authn.ClientRoleViewer)
	assert.NoError(t, err)

	err = adminSvc.AddClient(deviceID, "device 1", authn.ClientRoleDevice)
	assert.NoError(t, err)

	err = adminSvc.AddClient(serviceID, "service 1", authn.ClientRoleService)
	assert.NoError(t, err)

	// there should be 6 clients
	profiles, err := adminSvc.GetProfiles()
	require.NoError(t, err)
	assert.Equal(t, 7, len(profiles))

	err = adminSvc.RemoveClient("user1")
	assert.NoError(t, err)
	err = adminSvc.RemoveClient("user1") // remove is idempotent
	assert.NoError(t, err)
	err = adminSvc.RemoveClient("user2")
	assert.NoError(t, err)
	err = adminSvc.RemoveClient(deviceID)
	assert.NoError(t, err)
	err = adminSvc.RemoveClient(serviceID)
	assert.NoError(t, err)

	profiles, err = adminSvc.GetProfiles()
	// admin+two accounts remaining (user 3 and 4)
	require.NoError(t, err)
	assert.Equal(t, 3, len(profiles))
}

// Create manage users
func TestAddRemoveClientsFail(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const adminID = "administrator-1"
	_, authnSvc, stopFn := startTestAuthnServices(defaultHash)
	adminSvc := authnSvc.GetAdminService()
	defer stopFn()

	// missing clientID should fail
	err := adminSvc.AddClient("", "user 1", authn.ClientRoleService)
	assert.Error(t, err)

	// a bad key is not an error
	err = adminSvc.AddClient("user2", "user 2", authn.ClientRoleViewer)
	assert.NoError(t, err)
}

func TestUpdateClientPassword(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var tu1ID = "tu1ID"
	var tuPass1 = "tuPass1"
	var tuPass2 = "tuPass2"
	const adminID = "administrator-1"

	_, authnSvc, stopFn := startTestAuthnServices(defaultHash)
	adminSvc := authnSvc.GetAdminService()
	userSvc := authnSvc.GetUserService()
	defer stopFn()
	err := adminSvc.AddClient(tu1ID, "user tu1", authn.ClientRoleViewer)
	require.NoError(t, err)
	err = adminSvc.SetPassword(tu1ID, tuPass1)
	require.NoError(t, err)

	sm := userSvc.GetSessionManager()
	err = sm.ValidatePassword(tu1ID, tuPass1)
	require.NoError(t, err)

	err = adminSvc.SetPassword(tu1ID, tuPass2)
	require.NoError(t, err)

	err = sm.ValidatePassword(tu1ID, tuPass1)
	require.Error(t, err)

	err = sm.ValidatePassword(tu1ID, tuPass2)
	require.NoError(t, err)
}

func TestUpdatePubKey(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var tu1ID = "tu1ID"
	var tu1Pass = "tu1Pass"

	_, authnSvc, stopFn := startTestAuthnServices(defaultHash)
	adminSvc := authnSvc.GetAdminService()
	userSvc := authnSvc.GetUserService()
	defer stopFn()

	// add user to test with. don't set the public key yet
	err := adminSvc.AddClient(tu1ID, "user tu1", authn.ClientRoleViewer)
	adminSvc.SetPassword(tu1ID, tu1Pass)
	require.NoError(t, err)
	//
	sm := userSvc.GetSessionManager()
	token, validUntil, err := sm.CreateToken(tu1ID, time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, validUntil)

	// update the public key
	privKey, pubKey := utils.NewKey(utils.KeyTypeECDSA)
	require.NotEmpty(t, privKey)
	profile, err := adminSvc.GetProfile(tu1ID)
	require.NoError(t, err)
	profile.PubKeyPem = utils.PublicKeyToPem(pubKey)
	err = adminSvc.UpdateProfile(tu1ID, profile)
	assert.NoError(t, err)

	// check result
	profile2, err := adminSvc.GetProfile(tu1ID)
	require.NoError(t, err)
	assert.Equal(t, profile.PubKeyPem, profile2.PubKeyPem)
}

func TestNewDeviceToken(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var tu1ID = "device1ID"
	var tu1Name = "device 1"

	const adminID = "administrator-1"
	_, authnSvc, stopFn := startTestAuthnServices(defaultHash)
	adminSvc := authnSvc.GetAdminService()
	userSvc := authnSvc.GetUserService()
	defer stopFn()

	// add device to test with and connect
	err := adminSvc.AddClient(tu1ID, tu1Name, authn.ClientRoleDevice)
	require.NoError(t, err)

	// get a new token
	sm := userSvc.GetSessionManager()
	token, _, err := sm.CreateToken(tu1ID, time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// login with new token
	clientID, _, _, err := sm.ValidateClient(tu1ID, token)
	require.NoError(t, err)
	require.Equal(t, tu1ID, clientID)
}

func TestUpdateProfile(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	// const adminID = "administrator-1"
	_, authnSvc, stopFn := startTestAuthnServices(defaultHash)
	adminSvc := authnSvc.GetAdminService()

	defer stopFn()

	// add user to test with and connect
	err := adminSvc.AddClient(tu1ID, tu1Name, authn.ClientRoleViewer)
	require.NoError(t, err)
	//tu1Key, _ := testServer.MsgServer.CreateKP()

	// client can update display name
	const newDisplayName = "new display name"
	profile, err := adminSvc.GetProfile(tu1ID)
	require.NoError(t, err)
	assert.Equal(t, authn.ClientRoleViewer, profile.Role)
	profile.DisplayName = newDisplayName
	err = adminSvc.UpdateProfile(tu1ID, profile)
	assert.NoError(t, err)

	// verify
	profile2, err := adminSvc.GetProfile(tu1ID)
	require.NoError(t, err)
	assert.Equal(t, newDisplayName, profile2.DisplayName)
	assert.Equal(t, authn.ClientRoleViewer, profile2.Role)
}

func TestUpdateProfileFail(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const adminID = "administrator-1"
	var tu1ID = "tu1ID"
	var tu1Name = "test user 1"

	_, authnSvc, stopFn := startTestAuthnServices(defaultHash)
	adminSvc := authnSvc.GetAdminService()
	defer stopFn()
	// add user to test with and connect
	err := adminSvc.AddClient(tu1ID, tu1Name, authn.ClientRoleViewer)
	require.NoError(t, err)

	// this fails as badclient doesn't exist
	err = adminSvc.UpdateProfile(adminID, authn.ClientProfile{ClientID: "badclient"})
	assert.Error(t, err)
}
