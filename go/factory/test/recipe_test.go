package factory_test

import (
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells/authn"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/cells/thing"
	consumer_recipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
	gatewayrecipe "github.com/hiveot/hivekit/go/factory/recipes/gateway"
	rcdevice_recipe "github.com/hiveot/hivekit/go/factory/recipes/rcdevice"
	standalonerecipe "github.com/hiveot/hivekit/go/factory/recipes/sadevice"
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
	testDevice, err := testenv.NewTestCounterThing("", nil)
	require.NoError(t, err)
	defer testDevice.Stop()

	// Start the cell chain with a standalone server that links to the test Thing
	f := factory_service.NewCellFactory(env, nil)
	defer f.Stop()
	deviceRecipe, err := standalonerecipe.NewStandAloneDeviceRecipe(f, testDevice)
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

	serverFactory := factory_service.NewCellFactory(env, HiveKitAllCells)
	serverChain, err := factory_service.NewChainFormation(
		serverFactory, DeviceServerRecipe, nil)

	require.NotNil(t, serverChain)
	require.NoError(t, err)
	defer serverFactory.Stop()
	serverURLs := serverFactory.GetConnectURLs()
	require.NotEmpty(t, serverURLs)
	// use the gateway URL for a direct connection without forms
	env.GatewayURL = serverURLs[0]

	// the server exposed thing handles the server requests
	mod, _ := serverFactory.NewCell(thing.ExposedThingCellType, true)
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
	clientFactory := factory_service.NewCellFactory(env, HiveKitAllCells)
	clientChain, err := factory_service.NewChainFormation(
		clientFactory, DeviceClientRecipe, nil)

	require.NotNil(t, clientChain)
	require.NoError(t, err)
	defer clientFactory.Stop()

	m2, err := clientFactory.NewCell(consumer.ConsumerCellType, true)
	assert.NoError(t, err)
	co := m2.(*consumer.Consumer)
	var propValue string
	err = co.ReadProperty(thingID, "fortytwo", &propValue)
	assert.NoError(t, err)
	assert.NotEmpty(t, propValue)

}

// Test the gateway recipe with rcdevice and consumer recipes.
// This starts the gateway, connects an RC device, and let a consumer read its properties.
// TODO: use slots to use different protocols
func TestGatewayRecipe(t *testing.T) {
	fmt.Printf("---Test: %s %s---\n", t.Name(), testProtocol)
	const managerClientID = "manager1"
	const rcdeviceClientID = "rc1"

	// 1. Create the gateway recipe for admin and rc device connections
	gwenv := api.NewHiveEnvironment(testDir, false)
	gwenv.RpcTimeout = time.Minute * 3
	gwenv.HttpsPort = testPort
	gw, gwFactory, err := gatewayrecipe.NewGatewayRecipe(gwenv)
	require.NoError(t, err)
	gw.Start()
	defer gw.Stop()

	// 2. Create a manager using consumer recipe, and connect using token auth
	caCert, _ := gwenv.GetCACert()
	_, _, err = gw.AddAccount(managerClientID, "Manager 1", authn.ClientRoleManager, true, false)
	require.NoError(t, err)
	// env will load the client auth token
	coenv := api.NewHiveEnvironment(testDir, false)
	coenv.RpcTimeout = time.Minute * 3
	coenv.SetCACert(caCert)
	coenv.ClientID = managerClientID
	// force direct connection with the server
	coenv.GatewayURL = gwFactory.GetConnectURLs()[0]
	cor, _, err := consumer_recipe.NewConsumerRecipe(coenv, false)
	require.NoError(t, err)
	cor.Start()
	defer cor.Stop()

	// 3. Create a counter device using the RCDeviceRecipe and certificate with client cert
	_, _, err = gw.AddAccount(rcdeviceClientID, "device 1", authn.ClientRoleDevice, false, true)
	require.NoError(t, err)
	rcenv := api.NewHiveEnvironment(testDir, false)
	// no discovery, point to the server
	rcenv.RpcTimeout = time.Minute * 3
	rcenv.SetCACert(caCert)
	rcenv.GatewayURL = gwFactory.GetConnectURLs()[0]
	rcenv.ClientID = rcdeviceClientID
	// device configuration
	cfg := &testenv.CounterConfig{
		AutoIncrement: false,
		ResetValue:    60,
	}
	counterThing, err := testenv.NewTestCounterThing("", cfg)
	counterThing.SetTimeout(rcenv.RpcTimeout)
	rcr, _, err := rcdevice_recipe.NewRCDeviceRecipe(rcenv, counterThing)
	require.NoError(t, err)
	rcr.Start()
	counterThing.Start()
	defer counterThing.Stop()
	defer rcr.Stop()

	// 4. TODO: Add a stand-alone device and have Consumer connect to it via the gateway.

	// 5. consumer reads test device properties.
	// The (consumer) router calls GetTD from the consumer recipe directory client;
	//
	props, err := cor.ReadAllProperties(counterThing.GetID())
	require.NoError(t, err)
	assert.NotEmpty(t, props)

	// 6. done
}
