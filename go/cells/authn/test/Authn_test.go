package authn_test

import (
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/authn"
	authn_service "github.com/hiveot/hivekit/go/cells/authn/service"
	certstest "github.com/hiveot/hivekit/go/cells/certs/test"
	"github.com/hiveot/hivekit/go/cells/transport/tlsserver"
	tls_server "github.com/hiveot/hivekit/go/cells/transport/tlsserver/server"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
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

// This test file sets up the environment for testing authn admin and user services.
// This starts the authn service with a http server for testing the http API
func startTestAuthnService(encryption string) (tp api.IHttpServer, authnSvc authn.IAuthnService, stopFn func()) {

	_ = os.RemoveAll(testDir)
	_ = os.MkdirAll(testDir, 0700)

	//--- create the authentication service ---

	// the password file to use
	passwordFile := path.Join(testDir, "test.passwd")

	authnConfig = authn.NewAuthnConfig(testDir, testDir)
	authnConfig.PasswordFile = passwordFile
	authnConfig.AdminTokenValidityDays = 1
	// authnConfig.DeviceTokenValidityDays = 1
	authnConfig.Encryption = encryption

	authnSvc, err := authn_service.StartAuthnService(authnConfig)
	if err != nil {
		panic("Error starting authn admin service:" + err.Error())
	}
	// the session manager is a type of authenticator that also checks for
	// sessions started with login.
	authenticator := authnSvc.GetSessionManager()

	// create the http api handler for authn user requests over http
	testCerts = certstest.CreateTestCertBundle(TestKeyType)
	cfg := tlsserver.NewTLSServerConfig(
		"localhost", serverPort,
		testCerts.ServerCert, testCerts.RootCAs, true)

	httpServer, err := tls_server.NewTLSServer(cfg, authenticator)

	if err != nil {
		panic("Unable to start http server: " + err.Error())
	}
	authnHttpMod := authn_service.NewAuthnUserHttpService(httpServer)
	_ = authnHttpMod.Start()
	authnHttpMod.SetRequestSink(authnSvc)

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
	httpServer, m, stopFn := startTestAuthnService(defaultHash)
	require.NotNil(t, m)
	require.NotNil(t, httpServer)

	//	expect an admin token to be created in the keys dir
	adminTokenPath := filepath.Join(authnConfig.KeysDir, api.DefaultAdminUserID+api.DefaultTokenFileSuffix)
	assert.FileExists(t, adminTokenPath)

	defer stopFn()
}
