package factory_test

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules/authn"
	certstest "github.com/hiveot/hivekit/go/modules/certs/test"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/modules/digitwin"
	factory_service "github.com/hiveot/hivekit/go/modules/factory/service"
	"github.com/hiveot/hivekit/go/modules/thing"
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

	env := api.NewAppEnvironment(testDir, false)
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

	// just test that the environment can be created and loaded
	env := api.NewAppEnvironment(testDir, false)
	// err := env.LoadConfig(&env)
	// if err != nil {
	// t.Errorf("Failed loading config: %s", err.Error())
	// }
	f := factory_service.NewModuleFactory(env, nil)
	require.NotNil(t, f)
	// f.Start(recipe)
	f.Stop()
}

// test with the server module table
func TestAuthentication(t *testing.T) {
	// just test that the environment can be created and loaded
	env := api.NewAppEnvironment(testDir, false)
	env.CaCert = testCerts.CaCert
	env.TLSCert = testCerts.ServerCert
	env.HttpsPort = testPort

	f := factory_service.NewModuleFactory(env, HiveKitModules)
	assert.NotNil(t, f)
	defer f.Stop()

	// a server typically needs a http server and authenticator
	authenticator := f.GetAuthenticator()
	assert.NotNil(t, authenticator)

	httpServer := f.GetHttpServer(true)
	require.NotNil(t, httpServer)
	httpAuth := httpServer.GetAuthenticator()
	assert.NotNil(t, httpAuth)

	// loading the authn module switches the factory to use it as authenticator
	m, err := f.StartModule(authn.AuthnServiceModuleType, true)
	require.NotNil(t, m)
	assert.NoError(t, err)

	// create a token using authn session manager. It should validate with http authenticator now.
	authnMod, ok := m.(authn.IAuthnService)
	require.True(t, ok)
	sm := authnMod.GetSessionManager()
	err = authnMod.AddClient("client1", "client 1", "some role")
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
	// just test that the environment can be created and loaded
	env := api.NewAppEnvironment(testDir, false)
	env.CaCert = testCerts.CaCert
	env.TLSCert = testCerts.ServerCert
	env.HttpsPort = testPort

	f := factory_service.NewModuleFactory(env, HiveKitModules)
	defer f.Stop()

	// clientRecipe := factoryrecipe.NewFactoryRecipe(AvailableModules, chain)
	// clientRecipe.ModuleChain = []string{}

	// load the digitwin module
	// this should start the directory and http server
	m, err := f.StartModule(digitwin.DigitwinModuleType, true)
	require.NoError(t, err)
	require.NotNil(t, m)
}

// test creating a client app and server app using the recipe
func TestClientServerRecipe(t *testing.T) {
	var thingID string = "thing1"

	env := api.NewAppEnvironment(testDir, false)
	env.CaCert = testCerts.CaCert
	env.ClientCert = testCerts.ClientCert
	env.TLSCert = testCerts.ServerCert
	env.HttpsPort = testPort

	serverFactory := factory_service.NewModuleFactory(env, HiveKitModules)
	serverChain := factory_service.NewChainRecipe(serverFactory, DeviceServerRecipe)
	err := serverChain.Start()
	require.NoError(t, err)
	defer serverFactory.Stop()
	env.ServerURL = serverFactory.GetConnectURL()

	// the server exposed thing handles the server requests
	mod, _ := serverFactory.StartModule(thing.ExposedThingModuleType, true)
	device := mod.(*thing.ExposedThing)
	device.SetAppRequestHook(func(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
		if req.ThingID == thingID {
			slog.Info("Received request", "name", req.Name)
			resp := req.CreateResponse("42", nil)
			return replyTo(resp)
		}
		return fmt.Errorf("unknown request")
	})

	// the client sends requests and receives responses
	clientFactory := factory_service.NewModuleFactory(env, HiveKitModules)
	clientChain := factory_service.NewChainRecipe(clientFactory, DeviceClientRecipe)
	err = clientChain.Start()
	require.NoError(t, err)
	defer clientFactory.Stop()

	m2, err := clientFactory.StartModule(consumer.ConsumerModuleType, true)
	assert.NoError(t, err)
	co := m2.(*consumer.Consumer)
	var propValue string
	err = co.ReadProperty(thingID, "fortytwo", &propValue)
	assert.NoError(t, err)
	assert.NotEmpty(t, propValue)

}
