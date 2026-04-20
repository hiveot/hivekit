package authn_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/authn/internal/service"
	authnpkg "github.com/hiveot/hivekit/go/modules/authn/pkg"
	certstest "github.com/hiveot/hivekit/go/modules/certs/test"
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	httpserverconfig "github.com/hiveot/hivekit/go/modules/transports/httpserver/config"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/require"
)

var testDir = path.Join(os.TempDir(), "hivekit", "authn-test")
var authnConfig authn.AuthnConfig
var defaultHash = authn.PWHASH_ARGON2id

var serverPort int = 9445
var testCerts certstest.TestCertBundle
var testClientID1 = "client1"
var rpcTimeout = time.Minute * 3

const TestKeyType = utils.KeyTypeED25519

const appID = "authn-test"

// TestMain creates a test environment
// Used for all test cases in this package
func TestMain(m *testing.M) {
	utils.SetLogging("info", "")
	res := m.Run()
	if res == 0 {
		_ = os.RemoveAll(testDir)
	}
	os.Exit(res)
}

// NewTestConsumer creates a new connected consumer client with the given ID.
// The transport server must be started first.
//
// This uses the clientID as password
// This panics if a client cannot be created
func NewTestConsumer(m *service.AuthnService, protocolType, serverURL, clientID string) (
	*clients.Consumer, transports.ITransportClient, string) {

	// ensure the client exists
	_ = m.AddClient(clientID, "client 1", authn.ClientRoleViewer)
	sm := m.GetSessionManager()
	token, validUntil, _ := sm.CreateToken(clientID, time.Minute)
	_ = validUntil
	co, cc, err := clients.NewConsumerConnection(
		appID, protocolType, serverURL, testCerts.CaCert, rpcTimeout)
	if err != nil {
		panic("Failed creating consumer connection: " + err.Error())
	}
	cc.ConnectWithToken(clientID, token)

	return co, cc, token
}

// This test file sets up the environment for testing authn admin and user services.
// This starts the authn module with a http server for testing the http API
func startTestAuthnModule(encryption string) (tp transports.IHttpServer, authnSvc *service.AuthnService, stopFn func()) {

	_ = os.RemoveAll(testDir)
	_ = os.MkdirAll(testDir, 0700)

	//--- create the authentication service ---

	// the password file to use
	passwordFile := path.Join(testDir, "test.passwd")

	authnConfig = authn.NewAuthnConfig(testDir, testDir)
	authnConfig.PasswordFile = passwordFile
	// authnConfig.AgentTokenValidityDays = 1
	authnConfig.Encryption = encryption

	authnSvc = service.NewAuthnService(authnConfig)
	err := authnSvc.Start()
	if err != nil {
		panic("Error starting authn admin service:" + err.Error())
	}

	// create the http api handler for authn user requests over http
	testCerts = certstest.CreateTestCertBundle(TestKeyType)
	cfg := httpserverconfig.NewConfig(
		"localhost", serverPort,
		testCerts.ServerCert, testCerts.CaCert, nil, true)
	httpServer := httpserver.NewHttpServerModule(cfg)
	err = httpServer.Start()
	if err != nil {
		panic("Unable to start http server: " + err.Error())
	}
	authnHttpMod := authnpkg.NewAuthnUserHttpService(httpServer)
	_ = authnHttpMod.Start()
	authnHttpMod.SetRequestSink(authnSvc.HandleRequest)

	// last, link the http server and validator to enable the protected routes and enable the
	// authn http endpoint.
	var authenticator transports.IAuthenticator = authnSvc.GetSessionManager()
	httpServer.SetAuthenticator(authenticator)

	return httpServer, authnSvc, func() {
		authnSvc.Stop()
		httpServer.Stop()

		// let background tasks finish
		time.Sleep(time.Millisecond * 100)
	}
}

// Start the authn service and list clients
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	// this creates the admin user key
	httpServer, m, stopFn := startTestAuthnModule(defaultHash)
	require.NotNil(t, m)
	require.NotNil(t, httpServer)
	defer stopFn()
}

// func TestLogout(t *testing.T) {
// 	t.Logf("---%s---\n", t.Name())

// 	srv, tpauthn, cancelFn := StartTransportModule(nil)
// 	_ = srv
// 	defer cancelFn()

// 	// check if this test still works with a valid login
// 	cc1, co1, token1 := NewTestConsumer(tpauthn, testClientID1)
// 	_ = cc1
// 	_ = co1
// 	defer co1.Stop()
// 	assert.NotEmpty(t, token1)

// 	// logout
// 	serverURL := srv.GetConnectURL()
// 	authnClient := authnclient.NewAuthnHttpClient(serverURL, certBundle.CaCert)
// 	authnClient.ConnectWithToken(testClientID1, token1)
// 	err := authnClient.Logout(token1)
// 	assert.NoError(t, err)

// 	//authenticator.Logout(cc1, "")
// 	//err := co1.Logout()
// 	t.Log(">>> Logged out, an unauthorized error is expected next.")

// 	// This causes Refresh to fail
// 	token2, err := authnClient.RefreshToken(token1)
// 	//token2, err := co1.RefreshToken(token1)
// 	assert.Error(t, err)
// 	assert.Empty(t, token2)
// }

//func TestBadLogin(t *testing.T) {
//	t.Logf("---%s---\n", t.Name())
//
//	srv, cancelFn := StartTransportServer(nil, nil)
//	defer cancelFn()
//
//	cc1, co1, _ := NewConsumer(testClientID1, srv.GetForm)
//
//	// check if this test still works with a valid login
//	token1, err := cc1.ConnectWithPassword(testClientID1)
//	assert.NoError(t, err)
//
//	// failed logins
//	t.Log("Expecting ConnectWithPassword to fail")
//	token2, err := cc1.ConnectWithPassword("bad-pass")
//	assert.Error(t, err)
//	assert.Empty(t, token2)
//
//	// can't refresh when no longer connected
//	t.Log("Expecting RefreshToken to fail")
//	token4, err := co1.RefreshToken(token1)
//	assert.Error(t, err)
//	assert.Empty(t, token4)
//
//	// disconnect should always succeed
//	cc1.Disconnect()
//
//	// bad client ID
//	t.Log("Expecting ConnectWithPassword('BadID') to fail")
//	cc2, _, _ := NewConsumer("badID", srv.GetForm)
//	token5, err := cc2.ConnectWithPassword(testClientID1)
//	assert.Error(t, err)
//	assert.Empty(t, token5)
//}

// func TestBadRefresh(t *testing.T) {
// 	t.Logf("---%s---\n", t.Name())
// 	srv, tpauthn, cancelFn := StartTransportModule(nil)
// 	defer cancelFn()
// 	cc1, co1, token1 := NewTestConsumer(tpauthn, testClientID1)
// 	_ = co1
// 	_ = token1
// 	defer cc1.Close()

// 	// set the token
// 	t.Log("Expecting SetBearerToken('bad-token') to fail")
// 	err := cc1.ConnectWithToken(testClientID1, "bad-token")
// 	require.Error(t, err)

// 	// reconnect with a valid token and connect with a bad client-id
// 	err = cc1.ConnectWithToken(testClientID1, token1)
// 	assert.NoError(t, err)

// 	serverURL := srv.GetConnectURL()
// 	authCl := authnclient.NewAuthnHttpClient(serverURL, certBundle.CaCert)
// 	authCl.ConnectWithToken(testClientID1, token1)
// 	validToken, err := authCl.RefreshToken(token1)
// 	//validToken, err := co1.RefreshToken(token1)
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, validToken)
// 	cc1.Close()
// }
