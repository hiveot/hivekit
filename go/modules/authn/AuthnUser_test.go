package authn_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/authn"
	authnclient "github.com/hiveot/hivekit/go/modules/authn/client"
	"github.com/hiveot/hivekit/go/modules/authn/server"
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	tlsclient "github.com/hiveot/hivekit/go/modules/transports/httpserver/client"
	"github.com/hiveot/hivekit/go/utils"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginRefresh(t *testing.T) {
	var user1ID = "user1ID"
	var tu1Pass = "tu1Pass"

	m, stopFn := startTestAuthnModule(defaultHash)
	defer stopFn()

	// add user to test with
	err := m.AddClient(user1ID, testClientID1, authn.ClientRoleViewer)
	require.NoError(t, err)

	err = m.SetPassword(user1ID, tu1Pass)
	require.NoError(t, err)

	token1, validUntil, err := m.Login(user1ID, tu1Pass)
	require.NoError(t, err)
	require.Greater(t, validUntil, time.Now())

	cid2, validUntil2, err := m.ValidateToken(token1)
	require.NoError(t, err)
	assert.Equal(t, user1ID, cid2)
	require.Equal(t, validUntil2, validUntil)

	// RefreshToken the token after a short delay
	token3, validUntil3, err := m.RefreshToken(user1ID, token1)
	require.NoError(t, err)
	require.NotEmpty(t, token3)

	// ValidateToken the new token
	cid4, validUntil4, err := m.ValidateToken(token3)
	assert.Equal(t, user1ID, cid4)
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
	// http clients can't detect a bad token until making requests to a protected route
	// assert.Error(t, err)

	// reconnect with a valid token and connect with a bad client-id
	err = cc1.ConnectWithToken(testClientID1, token1, nil)
	assert.NoError(t, err)

	authCl := authnclient.NewAuthnHttpClient(serverURL, testCerts.CaCert)
	err = authCl.ConnectWithToken(testClientID1, token1)
	assert.NoError(t, err)
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

	m, cancelFn := startTestAuthnModule(defaultHash)
	defer cancelFn()

	tp := direct.NewDirectTransport(user1ID, m)

	// add user to test with
	co := clients.NewConsumer("test")
	authCl := authnclient.NewAuthnUserMsgClient(co)
	authCl.SetRequestSink(tp.HandleRequest)

	err := m.AddClient(user1ID, tu1Name, authn.ClientRoleViewer)
	m.SetPassword(user1ID, "oldpass")
	require.NoError(t, err)

	// login should succeed
	_, _, err = m.Login(user1ID, "oldpass")
	require.NoError(t, err)

	// change password
	err = m.SetPassword(user1ID, "newpass")
	require.NoError(t, err)

	// login with old password should now fail
	//t.Log("an error is expected logging in with the old password")
	_, _, err = m.Login(user1ID, "oldpass")
	require.Error(t, err)

	// re-login with new password
	_, _, err = m.Login(user1ID, "newpass")
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
	err := srv.AddClient(user1ID, tu1Name, authn.ClientRoleViewer)
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

	m, cancelFn := startTestAuthnModule(defaultHash)
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

// // Test certificate based authentication
func TestAuthClientCert(t *testing.T) {

	m, cancelFn := startTestAuthnModule(defaultHash)
	defer cancelFn()

	// add user to test with. don't set the public key yet
	err := m.AddClient(testCerts.ClientID, "user 1", authn.ClientRoleViewer)
	serverAddress := m.GetConnectURL()
	urlParts, err := url.Parse(serverAddress)

	tlsClient := tlsclient.NewTLSClient(
		urlParts.Host, testCerts.ClientCert, testCerts.CaCert, 0)

	// client should be able to read its profile using just client cert as auth
	getProfilePath := server.HttpGetProfilePath
	outputRaw, status, err := tlsClient.Get(getProfilePath)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)

	var profile authn.ClientProfile
	jsoniter.Unmarshal(outputRaw, &profile)
	assert.Equal(t, testCerts.ClientID, profile.ClientID)

	// clients.NewTransportClient(testAddress, testCerts.ClientCert, testCerts.CaCert, 0)
	// authCl := authnclient.NewAuthnHttpClient(urlParts.Host, testCerts.CaCert)

	// clientCert := tlsClient.GetClientCertificate()
	// assert.NotNil(t, clientCert)

	// // verify service certificate against CA
	// caCertPool := x509.NewCertPool()
	// caCertPool.AddCert(testCerts.CaCert)
	// opts := x509.VerifyOptions{
	// 	Roots:     caCertPool,
	// 	KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	// }
	// cert, err := x509.ParseCertificate(clientCert.Certificate[0])
	// if err == nil {
	// 	_, err = cert.Verify(opts)
	// }
	// assert.NoError(t, err)

}
