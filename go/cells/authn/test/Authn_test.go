package authn_test

import (
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/authn"
	authn_httpuserservice "github.com/hiveot/hivekit/go/cells/authn/httpuserservice"
	authn_service "github.com/hiveot/hivekit/go/cells/authn/service"
	certstest "github.com/hiveot/hivekit/go/cells/certs/test"
	"github.com/hiveot/hivekit/go/cells/transport/tlsserver"
	tls_server "github.com/hiveot/hivekit/go/cells/transport/tlsserver/server"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var passwdDir = path.Join(os.TempDir(), "hivekit-test", "authn")
var certsDir = passwdDir
var defaultHash = authn.PWHASH_ARGON2id

// var authnConfig authn.AuthnConfig

const unpwFileName = "testunpwstore.passwd"

var unpwFilePath string

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	utils.SetLogging("info", "")
	_ = os.MkdirAll(passwdDir, 0700)

	// Connect without pw file
	unpwFilePath = path.Join(passwdDir, unpwFileName)
	_ = os.Remove(unpwFilePath)

	res := m.Run()
	if res == 0 {
		_ = os.RemoveAll(passwdDir)
	}
	os.Exit(res)
}

// This test file sets up the environment for testing authn admin and user services.
// This starts the authn service with a http server for testing the http API
func startTestAuthnServices(encryption string) (
	tp api.IHttpServer, authnSvc authn.IAuthnService, stopFn func()) {

	_ = os.RemoveAll(passwdDir)
	_ = os.MkdirAll(passwdDir, 0700)

	//--- create the authentication service ---

	authnSvc, err := authn_service.NewAuthnService(certsDir, passwdDir, true)
	if err != nil {
		panic("startTestAuthnServices failed: " + err.Error())
	}
	// the session manager is a type of authenticator that also checks for
	// sessions started with login.
	authenticator := authnSvc.GetUserService().GetSessionManager()

	// create the http api handler for authn user requests over http
	testCerts = certstest.CreateTestCertBundle(TestKeyType)
	cfg := tlsserver.NewTLSServerConfig(
		"localhost", serverPort,
		testCerts.ServerCert, testCerts.RootCAs, true)

	httpServer, err := tls_server.NewTLSServer(cfg, authenticator)

	if err != nil {
		panic("Unable to start http server: " + err.Error())
	}
	authnHttpMod := authn_httpuserservice.NewAuthnUserHttpService(
		authn.AuthnUserServiceDefaultThingID, httpServer)
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
	httpServer, authnSvc, stopFn := startTestAuthnServices(defaultHash)
	require.NotNil(t, authnSvc)
	require.NotNil(t, httpServer)

	//	expect an admin token to be created in the keys dir
	adminTokenPath := filepath.Join(certsDir, api.DefaultAdminUserID+api.DefaultTokenFileSuffix)
	assert.FileExists(t, adminTokenPath)

	defer stopFn()
}
