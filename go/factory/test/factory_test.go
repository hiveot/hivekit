package factory_test

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/authn"
	certstest "github.com/hiveot/hivekit/go/cells/certs/test"
	"github.com/hiveot/hivekit/go/cells/digitwin"
	factory_service "github.com/hiveot/hivekit/go/factory/service"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testDir = path.Join(os.TempDir(), "hivekit", "factory-test")
var testCerts = certstest.CreateTestCertBundle(utils.KeyTypeED25519)
var testPort = 12345

// TestMain creates a test environment
// Used for all test cases in this package
func TestMain(m *testing.M) {
	utils.SetLogging("info", "")
	_ = os.RemoveAll(testDir)
	res := m.Run()
	if res == 0 {
		_ = os.RemoveAll(testDir)
	}
	os.Exit(res)
}

func TestAppEnv(t *testing.T) {
	fmt.Printf("---Test: %s %s---\n", t.Name(), testProtocol)

	env := api.NewHiveEnvironment(testDir, false)
	env.HttpsPort = testPort
	if env.HomeDir != testDir {
		t.Errorf("Expected homeDir to be %s, got %s", testDir, env.HomeDir)
	}
	if env.BinDir != path.Join(testDir, "bin") {
		t.Errorf("Expected binDir to be %s, got %s", path.Join(testDir, "bin"), env.BinDir)
	}
	// if f.PluginsDir != path.Join(testDir, "plugins") {
	// t.Errorf("Expected pluginsDir to be %s, got %s", path.Join(testDir, "plugins"), f.PluginsDir)
	// }
	if env.CertsDir != path.Join(testDir, "certs") {
		t.Errorf("Expected certsDir to be %s, got %s", path.Join(testDir, "certs"), env.CertsDir)
	}
	if env.LogsDir != path.Join(testDir, "logs") {
		t.Errorf("Expected logsDir to be %s, got %s", path.Join(testDir, "logs"), env.LogsDir)
	}
}

func TestStartStop(t *testing.T) {
	fmt.Printf("---Test: %s %s---\n", t.Name(), testProtocol)

	// just test that the environment can be created and loaded
	env := api.NewHiveEnvironment(testDir, false)
	// err := env.LoadConfig(&env)
	// if err != nil {
	// t.Errorf("Failed loading config: %s", err.Error())
	// }
	f := factory_service.StartCellFactory(env, nil)
	require.NotNil(t, f)
	// f.Start(recipe)
	f.Stop()
}

// test authentication using the factory
func TestAuthentication(t *testing.T) {
	fmt.Printf("---Test: %s %s---\n", t.Name(), testProtocol)

	// just test that the environment can be created and loaded
	env := api.NewHiveEnvironment(testDir, false)
	env.SetCACert(testCerts.CaCert)
	env.SetServerCert(testCerts.ServerCert)
	env.HttpsPort = testPort

	f := factory_service.StartCellFactory(env, HiveKitAllCells)
	assert.NotNil(t, f)
	defer f.Stop()

	// a server typically needs a http server and authenticator
	authenticator := f.GetAuthenticator()
	assert.NotNil(t, authenticator)

	httpServer := f.GetHttpServer(true)
	require.NotNil(t, httpServer)
	httpAuth := httpServer.GetAuthenticator()
	assert.NotNil(t, httpAuth)

	// loading the authn service switches the factory to use it as authenticator
	m, err := f.StartCell(authn.AuthnServiceCellType, true)
	require.NotNil(t, m)
	assert.NoError(t, err)

	// create a token using authn session manager. It should validate with http authenticator now.
	authnSvc, ok := m.(authn.IAuthnService)
	require.True(t, ok)
	sm := authnSvc.GetSessionManager()
	_, err = authnSvc.GetProfile("client1")
	if err != nil {
		err = authnSvc.AddClient("client1", "client 1", "some role")
	}
	require.NoError(t, err)
	token, _, err := sm.CreateToken("client1", time.Minute)
	require.NoError(t, err)

	// the httpauthn uses the factory authenticator which is set by authn to its session manager
	clientID, issAt, validUnt, err := httpAuth.ValidateClient("client1", token)
	require.NoError(t, err)
	assert.Equal(t, "client1", clientID)
	assert.NotNil(t, issAt)
	assert.NotNil(t, validUnt)

}

// test creating the digital twin instance
func TestDigitwin(t *testing.T) {
	fmt.Printf("---Test: %s %s---\n", t.Name(), testProtocol)

	// just test that the environment can be created and loaded
	env := api.NewHiveEnvironment(testDir, false)
	env.SetCACert(testCerts.CaCert)
	env.SetServerCert(testCerts.ServerCert)
	env.HttpsPort = testPort

	f := factory_service.StartCellFactory(env, HiveKitAllCells)
	defer f.Stop()

	// clientRecipe := factoryrecipe.NewFactoryRecipe(AvailableCells, chain)
	// clientRecipe.CellChain = []string{}

	// load the digitwin service
	// this should start the directory and http server
	m, err := f.StartCell(digitwin.DigitwinCellType, true)
	require.NoError(t, err)
	require.NotNil(t, m)
}
