package factory_test

import (
	"fmt"
	"log/slog"
	"testing"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/cells/thing"
	standalonerecipe "github.com/hiveot/hivekit/go/factory/recipes/standalone"
	factory_service "github.com/hiveot/hivekit/go/factory/service"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testprotocol used in the recipe
var testProtocol = api.HiveotSseScProtocolType

var testProtocols = []string{
	api.HiveotSseScProtocolType,
	api.HiveotGrpcTcpProtocolType,
	api.HiveotGrpcUnixProtocolType,
	api.HiveotWebsocketProtocolType,
	api.WotWebsocketProtocolType,
	api.HttpBasicProtocolType,
}

// Run tests that contain a protocol client in the recipe
// TODO: use slot to inject the selected server protocol
func TestAllProtocols(t *testing.T) {
	for _, testProtocol = range testProtocols {
		// t.Run("TestStandaloneDeviceRecipe - "+testProtocol, TestStandaloneDeviceRecipe)
		// t.Run("TestClientServerRecipes - "+testProtocol, TestClientServerRecipes)
	}
}

// Test the stand-alone device recipe
func TestStandaloneDeviceRecipe(t *testing.T) {
	fmt.Printf("---Test: %s %s---\n", t.Name(), testProtocol)

	env := api.NewHiveEnvironment(testDir, false)
	env.HttpsPort = testPort
	utils.SetLogging("info", "")

	// run a test Thing that will receive requests
	testDevice, err := testenv.StartTestCounterThing("", nil)
	require.NoError(t, err)
	defer testDevice.Stop()

	// Start the cell chain with a standalone server that links to the test Thing
	f := factory_service.StartCellFactory(env, nil)
	defer f.Stop()
	deviceRecipe, err := standalonerecipe.StartStandAloneDeviceRecipe(f, testDevice)
	require.NoError(t, err)
	defer deviceRecipe.Stop()

}

// test creating a client app and server app using the recipe
// TODO: use slots to use different protocols
func TestClientServerRecipes(t *testing.T) {
	fmt.Printf("---Test: %s %s---\n", t.Name(), testProtocol)
	var thingID string = "thing1"

	env := api.NewHiveEnvironment(testDir, false)
	env.SetCACert(testCerts.CaCert)
	env.SetClientCert(testCerts.ClientCert)
	env.SetServerCert(testCerts.ServerCert)
	env.HttpsPort = testPort

	serverFactory := factory_service.StartCellFactory(env, HiveKitAllCells)
	serverChain, err := factory_service.StartChainFormation(
		serverFactory, DeviceServerRecipe, nil)

	require.NotNil(t, serverChain)
	require.NoError(t, err)
	defer serverFactory.Stop()
	serverURLs := serverFactory.GetConnectURLs()
	require.NotEmpty(t, serverURLs)
	env.ServerURL = serverURLs[0]

	// the server exposed thing handles the server requests
	mod, _ := serverFactory.StartCell(thing.ExposedThingCellType, true)
	eThing := mod.(*thing.ExposedThing)
	eThing.SetAppRequestHook(func(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
		if req.ThingID == thingID {
			slog.Info("Received request", "name", req.Name)
			resp := req.CreateResponse("42", nil)
			return replyTo(resp)
		}
		return fmt.Errorf("unknown request")
	})

	// the client sends requests and receives responses
	clientFactory := factory_service.StartCellFactory(env, HiveKitAllCells)
	clientChain, err := factory_service.StartChainFormation(
		clientFactory, DeviceClientRecipe, nil)

	require.NotNil(t, clientChain)
	require.NoError(t, err)
	defer clientFactory.Stop()

	m2, err := clientFactory.StartCell(consumer.ConsumerCellType, true)
	assert.NoError(t, err)
	co := m2.(*consumer.Consumer)
	var propValue string
	err = co.ReadProperty(thingID, "fortytwo", &propValue)
	assert.NoError(t, err)
	assert.NotEmpty(t, propValue)

}
