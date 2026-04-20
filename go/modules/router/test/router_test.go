package router_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/modules/authn"
	certstest "github.com/hiveot/hivekit/go/modules/certs/test"
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/modules/directory"
	directorypkg "github.com/hiveot/hivekit/go/modules/directory/pkg"
	"github.com/hiveot/hivekit/go/modules/router"
	routerpkg "github.com/hiveot/hivekit/go/modules/router/pkg"
	"github.com/hiveot/hivekit/go/modules/transports"
	httpserverconfig "github.com/hiveot/hivekit/go/modules/transports/httpserver/config"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var storageDir = path.Join(os.TempDir(), "hivekit", "router-test")

var testDevicePort = 9993
var certsBundle = certstest.CreateTestCertBundle(utils.KeyTypeED25519)
var testAuthn = testenv.NewTestAuthenticator()

const rpcTimeout = time.Minute * 3
const testRouterID = "router1"

// const serverType = transports.HiveotWebsocketProtocolType

// const serverType = transports.HiveotSseScProtocolType

// const serverType = transports.WotHttpBasicProtocolType

const serverType = transports.ProtocolTypeWotWebsocket

// the test directory that holds this td. http server is not needed

// create a virtual test device
// this handles readallproperties requests
func startTestDevice(agentID string, thingID string) (testDevice *testenv.TestDevice) {

	testAuthn.AddClient(testRouterID, "", authn.ClientRoleManager)

	// create a test device with server
	cfg := httpserverconfig.NewConfig(
		certsBundle.ServerAddr, testDevicePort,
		certsBundle.ServerCert, certsBundle.CaCert, testAuthn, true)

	var testTM *td.TD = td.NewTD(thingID, "test device", vocab.Device)
	testTM.AddPropertyAsString("property-1", "Property 1", "New and improved")

	testDevice = testenv.NewTestDevice(cfg, agentID, testTM, serverType)
	err := testDevice.Start()
	if err != nil {
		panic("failed starting test device")
	}

	return testDevice
}

// Setup a consumer that uses the router to connect to devices
func SetupConsumerWithRouter() (
	routerMod router.IRouterService,
	dirMod directory.IDirectoryServer,
	co *clients.Consumer) {

	// setup the consumer side: directory, router and consumer
	// register the device TD in the directory for use by the router
	dirMod = directorypkg.NewDirectoryService("", storageDir, nil)
	err := dirMod.Start()
	if err != nil {
		panic("SetupConsumerWithRouter: Directory.Start: " + err.Error())
	}
	// defer testDirMod.Stop()
	// err = testDirMod.CreateThing(agentID, deviceTDJson)
	// require.NoError(t, err)

	// the router uses the TD to connect to the device.
	// this doesn't actually need a directory. GetTD could also simply return the device TD.
	routerMod = routerpkg.NewRouterService(
		storageDir, dirMod.GetTD, nil, certsBundle.CaCert)
	routerMod.SetTimeout(rpcTimeout)
	err = routerMod.Start()
	if err != nil {
		panic("SetupConsumerWithRouter: Router.Start: " + err.Error())
	}

	// defer routerMod.Stop()
	// to connect to the device, credentials are needed
	// FIXME: testAuthn does not properly test credentials. Use authn
	// token, _, _ := testAuthn.CreateToken(testRouterID, time.Minute)
	// routerMod.AddThingCredential(thingID1, clientID, token)

	// a consumer links to the router and subscribes to the device
	// note for the purpose of this test the router can run on the client
	consumer := clients.NewConsumer("")
	consumer.SetRequestSink(routerMod.HandleRequest)
	routerMod.SetNotificationSink(consumer.HandleNotification)
	err = consumer.Start()
	if err != nil {
		panic("SetupConsumerWithRouter: Consumer.Start: " + err.Error())
	}
	return routerMod, dirMod, consumer
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

	var testDirMod = directorypkg.NewDirectoryService("", "", nil)
	err := testDirMod.Start()
	require.NoError(t, err)
	// test no cred store
	m := routerpkg.NewRouterService("", testDirMod.GetTD, nil, certsBundle.CaCert)
	m.SetTimeout(rpcTimeout)
	err = m.Start()
	require.NoError(t, err)
	defer m.Stop()
}

// connect to a test device and subscribe to events
// in this setup the device runs a server and the router lives on the client.
func TestReadDeviceProperties(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const thingID1 = "thing-1"
	const agentID = "agent-1"
	const event1Name = "event1"
	const clientID = "client1"

	// Setup the test device with server and a TD
	// The test device runs a server. The router will have to match its security as
	// included in its TD
	testDevice := startTestDevice(agentID, thingID1)
	defer testDevice.Stop()
	testDevice.SetRequestHook(func(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
		testTD := testDevice.GetTD()
		if req.Operation == td.OpReadAllProperties {
			props := make(map[string]any)
			for name, aff := range testTD.Properties {
				props[name] = aff.Title
			}
			resp := req.CreateResponse(props, nil)
			return replyTo(resp)
		} else {
			resp := req.CreateResponse(nil, fmt.Errorf("unsupported op '%s'", req.Operation))
			return replyTo(resp)
		}
	})

	// setup the consumer with the router module
	routerMod, dirMod, co := SetupConsumerWithRouter()
	defer dirMod.Stop()
	defer routerMod.Stop()

	testTD := testDevice.GetTD()
	deviceTDJson, err := td.MarshalTD(testTD)
	err = dirMod.CreateThing(agentID, deviceTDJson)
	require.NoError(t, err)

	// to connect to the device, credentials are needed
	token, _, _ := testAuthn.CreateToken(testRouterID, time.Minute)
	routerMod.AddThingCredential(thingID1, clientID, token, td.SecSchemeBearer)

	// this should cause the router to connect to the device
	values, err := co.ReadAllProperties(thingID1)
	assert.NoError(t, err)
	assert.NotEmpty(t, values)
}

// connect to a test device and subscribe to events
// in this setup the device runs a server and the router lives on the client.
func TestSubscribeToDevice(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const thingID1 = "thing-1"
	const agentID = "agent-1"
	const event1Name = "event1"
	const clientID = "client1"
	var event1Value string = "value1"
	var rxValue string

	// Setup the test device with server and a TD
	// The test device runs a server. The router will have to match its security as
	// included in its TD
	testDevice := startTestDevice(agentID, thingID1)
	defer testDevice.Stop()

	// setup the consumer side: directory, router and consumer
	// register the device TD in the directory for use by the router
	var testDirMod = directorypkg.NewDirectoryService("", "", nil)
	err := testDirMod.Start()
	require.NoError(t, err)
	defer testDirMod.Stop()
	deviceTDJson, _ := td.MarshalTD(testDevice.GetTD())
	err = testDirMod.CreateThing(agentID, deviceTDJson)
	require.NoError(t, err)

	// the router uses the TD to connect to the device.
	// this doesn't actually need a directory. GetTD could also simply return the device TD.
	routerMod := routerpkg.NewRouterService(storageDir, testDirMod.GetTD, nil, certsBundle.CaCert)
	routerMod.SetTimeout(rpcTimeout)
	err = routerMod.Start()
	require.NoError(t, err)
	defer routerMod.Stop()
	// to connect to the device, credentials are needed
	// FIXME: testAuthn does not properly test credentials. Use authn
	token, _, _ := testAuthn.CreateToken(testRouterID, time.Minute)
	routerMod.AddThingCredential(thingID1, clientID, token, td.SecSchemeBearer)

	// a consumer links to the router and subscribes to the device
	// note for the purpose of this test the router can run on the client
	consumer := clients.NewConsumer("")
	consumer.SetRequestSink(routerMod.HandleRequest)
	routerMod.SetNotificationSink(consumer.HandleNotification)
	err = consumer.Start()
	assert.NoError(t, err)
	// this should cause the router to connect to the device
	err = consumer.Subscribe(thingID1, "")
	assert.NoError(t, err)

	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*60)
	consumer.SetNotificationHook(func(notif *msg.NotificationMessage) {
		assert.Equal(t, event1Name, notif.Name)
		err = notif.Decode(&rxValue)
		assert.NoError(t, err)
		cancelFn()
	})

	// the device publishes an event
	testDevice.Agent.PubEvent(thingID1, event1Name, event1Value)
	<-ctx.Done()
	cancelFn()
	assert.Equal(t, event1Value, rxValue)

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
	routerMod := routerpkg.NewRouterService(storageDir, nil, nil, nil)
	routerMod.SetTimeout(rpcTimeout)
	err := routerMod.Start()
	require.NoError(t, err)

	hasCred := routerMod.HasThingCredentials(thingID1)
	assert.False(t, hasCred)

	routerMod.AddThingCredential(thingID1, clientID, clientCred, thingScheme)

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
