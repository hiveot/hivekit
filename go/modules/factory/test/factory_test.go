package factory_test

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules/authn"
	certstest "github.com/hiveot/hivekit/go/modules/certs/test"
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/modules/digitwin"
	factory "github.com/hiveot/hivekit/go/modules/factory"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	factoryrecipe "github.com/hiveot/hivekit/go/modules/factory/recipe"
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

	f := factory.NewAppEnvironment(testDir, false)
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
	env := factory.NewAppEnvironment(testDir, false)
	err := env.LoadConfig(&env)
	if err != nil {
		t.Errorf("Failed loading config: %s", err.Error())
	}
	f := factorypkg.NewModuleFactory(env, nil)
	require.NotNil(t, f)
	// f.Start(recipe)
	f.StopAll()
}

// test with the server module table
func TestAuthentication(t *testing.T) {
	// just test that the environment can be created and loaded
	env := factory.NewAppEnvironment(testDir, false)
	env.CaCert = testCerts.CaCert
	env.ServerCert = testCerts.ServerCert

	f := factorypkg.NewModuleFactory(env, RecipeModules)
	assert.NotNil(t, f)
	defer f.StopAll()

	// a server typically needs a http server and authenticator
	authenticator := f.GetAuthenticator()
	assert.NotNil(t, authenticator)

	httpServer := f.GetHttpServer()
	assert.NotNil(t, httpServer)
	httpAuth := httpServer.GetAuthenticator()
	assert.NotNil(t, httpAuth)

	// loading the authn module switches the factory to use it as authenticator
	m, err := f.GetModule(authn.AuthnModuleType)
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
	clientID, issAt, validUnt, err := httpAuth.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "client1", clientID)
	assert.NotNil(t, issAt)
	assert.NotNil(t, validUnt)

}

// test creating the digital twin instance
func TestDigitwin(t *testing.T) {
	// just test that the environment can be created and loaded
	env := factory.NewAppEnvironment(testDir, false)
	env.CaCert = testCerts.CaCert
	env.ServerCert = testCerts.ServerCert

	f := factorypkg.NewModuleFactory(env, RecipeModules)
	defer f.StopAll()

	// clientRecipe := factoryrecipe.NewFactoryRecipe(AvailableModules, chain)
	// clientRecipe.ModuleChain = []string{}

	// load the digitwin module
	// this should start the directory and http server
	m, err := f.GetModule(digitwin.DigitwinModuleType)
	require.NotNil(t, m)
	assert.NoError(t, err)
}

// test creating a client app and server app using the recipe
func TestClientServerRecipe(t *testing.T) {
	var thingID string = "thing1"

	env := factory.NewAppEnvironment(testDir, false)
	env.CaCert = testCerts.CaCert
	env.ServerCert = testCerts.ServerCert

	serverFactory := factorypkg.NewModuleFactory(env, RecipeModules)
	serverChain := factoryrecipe.NewFactoryRecipe(RecipeModules, TestDeviceServerChain)
	err := serverChain.Start(serverFactory)
	require.NoError(t, err)
	defer serverFactory.StopAll()

	clientFactory := factorypkg.NewModuleFactory(env, RecipeModules)
	clientChain := factoryrecipe.NewFactoryRecipe(RecipeModules, TestDeviceClientChain)
	err = clientChain.Start(clientFactory)
	require.NoError(t, err)
	defer clientFactory.StopAll()

	// the agent handles the server requests
	ag, _ := clientFactory.GetModule(clients.AgentModuleType)
	ag.SetRequestHook(func(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
		if req.ThingID == thingID {
			slog.Info("Received request", "name", req.Name)
			resp := req.CreateResponse("42", nil)
			return replyTo(resp)
		}
		return fmt.Errorf("unknown request")
	})

	// the client sends requests and receives responses
	// FIXME: this fails
	// m, _ := clientFactory.GetModule(clients.ConsumerModuleType)
	// co := m.(*clients.Consumer)
	// props, err := co.ReadAllProperties(thingID)
	// assert.NoError(t, err)
	// assert.NotEmpty(t, props)

	// send a request
}
