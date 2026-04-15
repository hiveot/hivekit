package factory_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/factory"
	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"
	certstest "github.com/hiveot/hivekit/go/modules/certs/test"
	digitwinapi "github.com/hiveot/hivekit/go/modules/digitwin/api"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testDir = path.Join(os.TempDir(), "hivekit", "factory-test")
var testCerts = certstest.CreateTestCertBundle(utils.KeyTypeED25519)

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

func TestAppEnv(t *testing.T) {

	f := factoryapi.NewAppEnvironment(testDir, false)
	if f.HomeDir != testDir {
		t.Errorf("Expected homeDir to be %s, got %s", testDir, f.HomeDir)
	}
	if f.BinDir != path.Join(testDir, "bin") {
		t.Errorf("Expected binDir to be %s, got %s", path.Join(testDir, "bin"), f.BinDir)
	}
	if f.PluginsDir != path.Join(testDir, "plugins") {
		t.Errorf("Expected pluginsDir to be %s, got %s", path.Join(testDir, "plugins"), f.PluginsDir)
	}
	if f.CertsDir != path.Join(testDir, "certs") {
		t.Errorf("Expected certsDir to be %s, got %s", path.Join(testDir, "certs"), f.CertsDir)
	}
	if f.LogsDir != path.Join(testDir, "logs") {
		t.Errorf("Expected logsDir to be %s, got %s", path.Join(testDir, "logs"), f.LogsDir)
	}
}

func TestStartStop(t *testing.T) {
	_ = os.RemoveAll(testDir)

	// just test that the environment can be created and loaded
	env := factoryapi.NewAppEnvironment(testDir, false)
	err := env.LoadConfig(&env)
	if err != nil {
		t.Errorf("Failed loading config: %s", err.Error())
	}
	f := factory.NewModuleFactory(env, nil)
	assert.NotNil(t, f)
	// f.Start()
	// f.Stop()
}

// test with the server module table
func TestGetAuthenticator(t *testing.T) {
	// just test that the environment can be created and loaded
	env := factoryapi.NewAppEnvironment(testDir, false)
	env.CaCert = testCerts.CaCert
	env.ServerCert = testCerts.ServerCert

	f := factory.NewModuleFactory(env, ServerModuleTable)
	assert.NotNil(t, f)

	// a server typically needs a http server and authenticator
	authenticator := f.GetAuthenticator()
	assert.NotNil(t, authenticator)

	httpServer := f.GetHttpServer()
	assert.NotNil(t, httpServer)
	httpAuth := httpServer.GetAuthenticator()
	assert.NotNil(t, httpAuth)

	// loading the authn module switches the factory to use it as authenticator
	m, err := f.GetModule(authnapi.AuthnModuleType)
	require.NotNil(t, m)
	assert.NoError(t, err)

	// create a token using authn session manager. It should validate with http authenticator now.
	authnMod, ok := m.(authnapi.IAuthnService)
	require.True(t, ok)
	sm := authnMod.GetSessionManager()
	err = authnMod.AddClient("client1", "client 1", "some role")
	require.NoError(t, err)
	token, _, err := sm.CreateToken("client1", time.Minute)
	require.NoError(t, err)

	// the httpauthn uses the factory authenticator which is set by authn to its session manager
	clientID, issAt, validUnt, err := httpAuth.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "client1", clientID)
	assert.NotNil(t, issAt)
	assert.NotNil(t, validUnt)
}

// test with the digital twin factory
func TestDigitwin(t *testing.T) {
	// just test that the environment can be created and loaded
	env := factoryapi.NewAppEnvironment(testDir, false)
	env.CaCert = testCerts.CaCert
	env.ServerCert = testCerts.ServerCert

	f := factory.NewModuleFactory(env, ServerModuleTable)

	// load the digitwin module
	// this should start the directory and http server
	m, err := f.GetModule(digitwinapi.DigitwinModuleType)
	require.NotNil(t, m)
	assert.NoError(t, err)
}
