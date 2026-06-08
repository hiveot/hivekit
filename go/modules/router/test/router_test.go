package router_test

import (
	"context"
	"log/slog"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/modules/authn"
	certstest "github.com/hiveot/hivekit/go/modules/certs/test"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/modules/directory"
	directorypkg "github.com/hiveot/hivekit/go/modules/directory/pkg"
	"github.com/hiveot/hivekit/go/modules/router"
	routerpkg "github.com/hiveot/hivekit/go/modules/router/pkg"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/httptransport"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var storageDir = path.Join(os.TempDir(), "hivekit", "router-test")

var testDevicePort = 9993
var certsBundle = certstest.CreateTestCertBundle(utils.KeyTypeED25519)
var testAuthn = testenv.NewTestAuthenticator()

const rpcTimeout = time.Minute * 3 // allow for debugging breakpoints
const testRouterID = "router1"

// const serverType = transport.ProtocolTypeHiveotWebsocket

const serverType = transport.ProtocolTypeHiveotSsesc

// const serverType = transport.ProtocolTypeWotHttpBasic

// const serverType = transport.ProtocolTypeWotWebsocket

// the test directory that holds this td. http server is not needed

// create a virtual test device with the test environment.
// this device handles read requests using its agent.
func startTestDevice(agentID string) (testDevice *testenv.TestDevice) {

	testAuthn.AddClient(testRouterID, "", authn.ClientRoleManager)

	// create a test device with server
	cfg := httptransport.NewConfig(
		certsBundle.ServerAddr, testDevicePort,
		certsBundle.ServerCert, certsBundle.CaCert, true)

	var testTM *td.TD = td.NewTD(agentID, "test device", vocab.Device)
	testTM.AddPropertyAsString("property-1", "Property 1", "New and improved")

	testDevice = testenv.NewTestDevice(cfg, agentID, testAuthn, testTM, serverType)
	err := testDevice.Start()
	if err != nil {
		panic("failed starting test device")
	}

	return testDevice
}

// Setup a consumer that uses the router to connect to devices
func SetupConsumerWithRouter() (
	co *consumer.Consumer,
	routerMod router.IRouterService,
	dirMod directory.IDirectoryService,
) {

	// setup the consumer side: directory, router and consumer
	// register the device TD in the directory for use by the router
	dirMod = directorypkg.NewDirectoryMsgServer("", storageDir, nil, nil)
	err := dirMod.Start()
	if err != nil {
		panic("SetupConsumerWithRouter: Directory.Start: " + err.Error())
	}

	// the router uses the TD to connect to the device.
	// this doesn't actually need a directory. GetTD could also simply return the device TD.
	routerMod = routerpkg.NewRouterService(
		storageDir, dirMod.GetTD, nil, certsBundle.CaCert, rpcTimeout)

	err = routerMod.Start()
	if err != nil {
		panic("SetupConsumerWithRouter: Router.Start: " + err.Error())
	}

	// a consumer links to the router and subscribes to the device
	// note for the purpose of this test the router can run on the client
	co = consumer.NewConsumer(nil)
	co.SetTimeout(rpcTimeout)
	co.SetRequestSink(routerMod)
	routerMod.SetNotificationSink(co)
	err = co.Start()
	if err != nil {
		panic("SetupConsumerWithRouter: Consumer.Start: " + err.Error())
	}
	return co, routerMod, dirMod
}

// TestMain create a test folder for certificates and private key
func TestMain(m *testing.M) {
	utils.SetLogging("info", "")

	result := m.Run()
	if result != 0 {
		println("Test failed with code:", result)
	} else {
	}

	os.Exit(result)
}

// Generic directory store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	var testDirMod = directorypkg.NewDirectoryMsgServer("", "", nil, nil)
	err := testDirMod.Start()
	require.NoError(t, err)
	// test no cred store
	m := routerpkg.NewRouterService("", testDirMod.GetTD, nil, certsBundle.CaCert, rpcTimeout)
	err = m.Start()
	require.NoError(t, err)
	defer m.Stop()
}

func TestCredStore(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const thingID1 = "thing-1"
	const clientID = "client1"
	const clientCred = "secret"
	const thingScheme = td.SecSchemeBearer

	os.RemoveAll(storageDir)

	// the router uses the TD to connect to the device.
	// this doesn't actually need a directory. GetTD could also simply return the device TD.
	routerMod := routerpkg.NewRouterService(storageDir, nil, nil, nil, rpcTimeout)
	err := routerMod.Start()
	require.NoError(t, err)

	hasCred := routerMod.HasThingCredentials(thingID1)
	assert.False(t, hasCred)

	routerMod.AddDeviceCredential(thingID1, clientID, clientCred, thingScheme)

	hasCred = routerMod.HasThingCredentials(thingID1)
	assert.True(t, hasCred)

	routerMod.Stop()

	// restarting the router module should retain the credentials
	err = routerMod.Start()
	require.NoError(t, err)

	hasCred = routerMod.HasThingCredentials(thingID1)
	assert.True(t, hasCred)
	routerMod.Stop()
}

// connect to a test device and subscribe to events
// in this setup the device runs a server and the router lives on the client.
func TestReadDeviceProperties(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent-1"
	const clientID = "router1"
	const prop1Name = "prop1"
	const prop1Value = "value1"

	// Setup the test device with server and a TD
	// The test device is a runs a server that passes requests to its agent.
	// The router will have to match its security as described in its TD
	// The agent of the test device handles read requests
	testDevice := startTestDevice(agentID)
	defer testDevice.Stop()
	testDevice.Agent.PubProperty(agentID, prop1Name, prop1Value)

	// setup the consumer with the router module
	co, routerMod, dirMod := SetupConsumerWithRouter()
	defer dirMod.Stop()
	defer routerMod.Stop()

	testTD := testDevice.GetTD()
	deviceTDJson, err := td.MarshalTD(testTD)
	err = dirMod.CreateThing(agentID, deviceTDJson)
	require.NoError(t, err)

	// to connect to the device, credentials are needed
	token, _, _ := testAuthn.CreateToken(testRouterID, rpcTimeout)
	routerMod.AddDeviceCredential(agentID, clientID, token, td.SecSchemeBearer)

	// this should cause the router to connect to the device
	values, err := co.ReadAllProperties(agentID)
	assert.NoError(t, err)
	assert.NotEmpty(t, values)
}

// connect to a test device and subscribe to events
// in this setup the device runs a server and the router lives on the client.
func TestSubscribeReconnectToDevice(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent-1"
	const clientID = "router1"
	event1Name := "event1"
	event1Value := "value1"
	event2Value := "value2"
	prop1Name := "prop1"
	prop1Value1 := "prop1value1"
	prop1Value2 := "prop1value2"
	var rxValue = new(atomic.Value)
	// rxChan := make(chan *msg.NotificationMessage, 1)

	// Setup the test device with server and a TD
	// The test device runs a server. The router will have to match its security as
	// included in its TD
	testDevice := startTestDevice(agentID)
	defer testDevice.Stop()

	// setup the consumer side: directory, router and consumer
	// register the device TD in the directory for use by the router
	var testDirMod = directorypkg.NewDirectoryMsgServer("", "", nil, nil)
	err := testDirMod.Start()
	require.NoError(t, err)
	defer testDirMod.Stop()
	deviceTDJson, _ := td.MarshalTD(testDevice.GetTD())
	err = testDirMod.CreateThing(agentID, deviceTDJson)
	require.NoError(t, err)

	// the router uses the TD to connect to the device.
	// this doesn't actually need a directory. GetTD could also simply return the device TD.
	routerMod := routerpkg.NewRouterService(storageDir, testDirMod.GetTD, nil, certsBundle.CaCert, rpcTimeout)
	err = routerMod.Start()
	require.NoError(t, err)
	defer routerMod.Stop()

	// to connect to the device, credentials are needed
	// FIXME: testAuthn does not properly test credentials. Use authn
	token, _, _ := testAuthn.CreateToken(testRouterID, rpcTimeout)
	routerMod.AddDeviceCredential(agentID, clientID, token, td.SecSchemeBearer)

	ctx, cancelFn := context.WithTimeout(context.Background(), rpcTimeout)

	// a consumer links to the router which connects to devices using device TDs
	co := consumer.NewConsumer(func(notif *msg.NotificationMessage) {
		if notif.Name == event1Name {
			var v1 string
			err = notif.Decode(&v1)
			rxValue.Store(v1)
			assert.NoError(t, err)
			cancelFn()
			// rxChan <- notif
		}
	})
	co.SetTimeout(rpcTimeout)
	co.SetRequestSink(routerMod)
	// notifications received are passed back to the consumer
	routerMod.SetNotificationSink(co)
	err = co.Start()
	assert.NoError(t, err)
	// this should cause the router to connect to the device using the device TD
	err = co.Subscribe(agentID, "")
	assert.NoError(t, err)

	// the device updates a property and event
	testDevice.Agent.PubProperty(agentID, prop1Name, prop1Value1)
	testDevice.Agent.PubEvent(agentID, event1Name, event1Value)
	<-ctx.Done()
	assert.Equal(t, event1Value, rxValue.Load())

	//--- phase 2 test subscription after auto reconnect

	// drop client connections
	slog.Info("--- breaking connections, expect a warning ---")
	testDevice.TransportServer.CloseAll()

	// reading properties should fail while auto-reconnect is ongoing
	slog.Info("---ReadAllProperties---")
	values, err := co.ReadAllProperties(agentID)
	assert.Error(t, err)

	// lets sleep to allow for reconnect
	time.Sleep(time.Second)

	// publish a property should now succeed
	testDevice.Agent.PubProperty(agentID, prop1Name, prop1Value2)

	values, err = co.ReadAllProperties(agentID)
	assert.NoError(t, err)

	time.Sleep(time.Millisecond) // time to receive
	assert.Equal(t, prop1Value2, values[prop1Name])

	// on reconnect, subscription should remain intact and event should be received
	testDevice.Agent.PubEvent(agentID, event1Name, event2Value)
	time.Sleep(time.Millisecond) // time to receive
	assert.Equal(t, event2Value, rxValue.Load())

}
