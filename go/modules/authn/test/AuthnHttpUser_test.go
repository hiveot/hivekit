package authn_test

import (
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/authn"
	authnpkg "github.com/hiveot/hivekit/go/modules/authn/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBadRefreshHttp(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	httpServer, svc, cancelFn := startTestAuthnModule(defaultHash)
	defer cancelFn()

	err := svc.AddClient(testClientID1, "client 1", authn.ClientRoleViewer)
	sm := svc.GetSessionManager()
	token1, _, err := sm.CreateToken(testClientID1, time.Minute)
	assert.NoError(t, err)

	serverURL := httpServer.GetConnectURL()
	authCl := authnpkg.NewUserAuthnHttpClient(serverURL, nil, testCerts.CaCert)
	defer authCl.Close()
	err = authCl.ConnectWithToken(testClientID1, token1)
	assert.NoError(t, err)
	// http clients can't detect a bad token until making requests to a protected route
	// assert.Error(t, err)

	// refresh validates the token provided in the connection
	token2, err := authCl.RefreshToken(token1)
	assert.NotEmpty(t, token2)
	assert.NoError(t, err)

	t.Log("*** Expecting SetBearerToken('bad-token') to fail ***")
	token3, err := authCl.RefreshToken("badToken")
	assert.Error(t, err)
	assert.Empty(t, token3)

}

func TestLogoutHttp(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	httpServer, svc, cancelFn := startTestAuthnModule(defaultHash)
	defer cancelFn()

	err := svc.AddClient(testClientID1, "client 1", authn.ClientRoleViewer)
	sm := svc.GetSessionManager()
	token1, _, err := sm.CreateToken(testClientID1, time.Minute)
	require.NoError(t, err)

	serverURL := httpServer.GetConnectURL()
	authnClient := authnpkg.NewUserAuthnHttpClient(serverURL, nil, testCerts.CaCert)
	err = authnClient.ConnectWithToken(testClientID1, token1)
	defer authnClient.Close()

	token2, err := authnClient.RefreshToken(token1)
	require.NoError(t, err)

	// logout
	err = authnClient.Logout(token2)
	assert.NoError(t, err)

	t.Log(">>> Logged out, an unauthorized error is expected next.")

	// This causes Refresh to fail
	token3, err := authnClient.RefreshToken(token1)
	assert.Error(t, err)
	assert.Empty(t, token3)

	token3, err = authnClient.RefreshToken(token2)
	assert.Error(t, err)
	assert.Empty(t, token3)
}

// Test certificate based authentication
func TestAuthClientCertHttp(t *testing.T) {

	httpServer, svc, cancelFn := startTestAuthnModule(defaultHash)
	defer cancelFn()

	// add user to test with. don't set the public key yet
	err := svc.AddClient(testCerts.ClientID, "user 1", authn.ClientRoleViewer)
	require.NoError(t, err)
	serverURL := httpServer.GetConnectURL()

	// client should be able to read its profile using just client cert as auth
	authCl := authnpkg.NewUserAuthnHttpClient(serverURL, testCerts.ClientCert, testCerts.CaCert)
	defer authCl.Close()

	profile, err := authCl.GetProfile()
	require.NoError(t, err)
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
