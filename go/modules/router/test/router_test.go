package router_test

import (
	"context"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/authn"
	certstest "github.com/hiveot/hivekit/go/modules/certs/test"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/modules/directory"
	directorypkg "github.com/hiveot/hivekit/go/modules/directory/pkg"
	"github.com/hiveot/hivekit/go/modules/router"
	routerpkg "github.com/hiveot/hivekit/go/modules/router/pkg"
	grpcpkg "github.com/hiveot/hivekit/go/modules/transport/grpc/pkg"
	httpbasicpkg "github.com/hiveot/hivekit/go/modules/transport/httpbasic/pkg"
	ssescpkg "github.com/hiveot/hivekit/go/modules/transport/ssesc/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/tlsserver"
	tlsserverpkg "github.com/hiveot/hivekit/go/modules/transport/tlsserver/pkg"
	wsspkg "github.com/hiveot/hivekit/go/modules/transport/wss/pkg"
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
const testConsumerID = "router1"

const serverType = api.ProtocolTypeHiveotGrpc

// const serverType = api.ProtocolTypeHiveotWebsocket

// const serverType = api.ProtocolTypeHiveotSsesc

// const serverType = api.ProtocolTypeWotHttpBasic

// const serverType = api.ProtocolTypeWotWebsocket

// the test directory that holds this td. http server is not needed

// create a chain with a virtual test device, a server and authenticator:
//
//	> authn -> http server - protocol server -> testdevice -> discovery server
//
// This device handles read requests and publishes notifications.
//
// Intended for testing client side / router connections
//
// The deviceID is the thingID of the device
func startTestServerDevice(deviceID string) (testDevice *testenv.TestDevice,
	tdoc *td.TD, transportServer api.ITransportServer, stopFn func()) {

	// 1. need a http server for serving the protocol and optionally discovery
	cfg := tlsserver.NewTLSServerConfig(
		certsBundle.ServerAddr, testDevicePort,
		certsBundle.ServerCert, certsBundle.CaCert, true)

	httpServer := tlsserverpkg.NewTLSServer(cfg, testAuthn)
	err := httpServer.Start()
	if err != nil {
		panic("startTestServer: failed to start http server")
	}

	// 2. Create a protocol server for receiving requests
	switch serverType {
	case api.ProtocolTypeWotHttpBasic:
		transportServer = httpbasicpkg.NewHttpBasicServer(httpServer)
	case api.ProtocolTypeHiveotGrpc:
		address := "unix://" + filepath.Join(storageDir, "grpc-server.sock")
		transportServer = grpcpkg.NewHiveotGrpcServer(
			address, cfg.ServerCert, cfg.CaCert, testAuthn, 0)
	case api.ProtocolTypeHiveotSsesc:
		transportServer = ssescpkg.NewSseScServer(httpServer, 0)
	case api.ProtocolTypeWotWebsocket:
		transportServer = wsspkg.NewWotWssServer(httpServer, 0)
	case api.ProtocolTypeHiveotWebsocket:
		transportServer = wsspkg.NewHiveotWssServer(httpServer, 0)
	}
	err = transportServer.Start()
	if err != nil {
		panic("startTestServerDevice: failed to start transport server " + serverType)
	}
	slog.Info("startTestServerDevice", "deviceID", deviceID, "serverType", serverType)
	// var testTM *td.TD = td.NewTD(deviceID, "test device", vocab.Device)
	// testTM.AddPropertyAsString("property-1", "Property 1", "New and improved")

	// testDevice = testenv.NewTestDevice(cfg, deviceID, testAuthn, testTM, serverType)

	// 3. Create the test device Thing
	testDevice = testenv.NewCounterDevice(deviceID, nil)
	testDevice.SetNotificationSink(transportServer)
	transportServer.SetRequestSink(testDevice)
	err = testDevice.Start()
	if err != nil {
		panic("startTestServerDevice: failed starting test device")
	}

	// Add the connection forms to the device TD
	tdJson := testDevice.GetTD()
	tdoc, _ = td.UnmarshalTD(tdJson)
	transportServer.AddTDSecForms(tdoc, false)

	return testDevice, tdoc, transportServer, func() {
		testDevice.Stop()
		transportServer.Stop()
		httpServer.Stop()
	}
}

// Setup a consumer that uses the router and directory to connect to devices
// The router has a credentials store for authentication
func SetupConsumerWithRouter(authn api.IAuthenticator) (
	co *consumer.Consumer,
	routerMod router.IRouterService,
	dirSvc directory.IDirectoryService,
) {

	// setup the consumer side: directory, router and consumer
	// register the device TD in the directory for use by the router
	dirSvc = directorypkg.NewDirectoryService("", storageDir, nil, nil)
	err := dirSvc.Start()
	if err != nil {
		panic("SetupConsumerWithRouter: Directory.Start: " + err.Error())
	}

	// the router uses the TD to connect to the device.
	// this doesn't actually need a directory. GetTD could also simply return the device TD.
	routerMod = routerpkg.NewRouterService(
		storageDir, dirSvc.GetTD, nil, certsBundle.CaCert, rpcTimeout)

	err = routerMod.Start()
	if err != nil {
		panic("SetupConsumerWithRouter: Router.Start: " + err.Error())
	}

	// A consumer links to the router and subscribes to the device.
	// For the purpose of this test the router runs client side.
	co = consumer.NewConsumer(routerMod, nil)
	co.SetTimeout(rpcTimeout)
	err = co.Start()
	if err != nil {
		panic("SetupConsumerWithRouter: Consumer.Start: " + err.Error())
	}
	return co, routerMod, dirSvc
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

	var testDirMod = directorypkg.NewDirectoryService("", "", nil, nil)
	err := testDirMod.Start()
	require.NoError(t, err)
	// test no cred store
	m := routerpkg.NewRouterService("", testDirMod.GetTD, nil, certsBundle.CaCert, rpcTimeout)
	err = m.Start()
	require.NoError(t, err)
	defer m.Stop()
}

func TestCredentialsStore(t *testing.T) {
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

// connect to a stand-alone test device and read its properties
func TestReadObserveDeviceProperties(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const deviceID = "device1"
	const clientID = "router1"
	const prop1Name = "prop1"
	const prop1Value = "value1"
	var notifCount atomic.Int32

	// Setup the test device with server and a TD
	// The test device is a runs a server that passes requests to its device.
	// The router will have to match its security as described in the device TD
	// The device of the test device handles read requests
	testDevice, device1TD, _, stopFn := startTestServerDevice(deviceID)
	defer stopFn()

	// when the device publishes an observable property it becomes available for querying
	testDevice.ExposedThing.PubProperty(deviceID, prop1Name, prop1Value, false)

	// setup the consumer with the router module and directory client or service
	co, routerMod, dirSvc := SetupConsumerWithRouter(testAuthn)
	defer dirSvc.Stop()
	defer routerMod.Stop()

	// test if subscribe works
	co.SetNotificationHook(func(notif *msg.NotificationMessage) {
		if notif.AffordanceType == msg.AffordanceTypeProperty {
			notifCount.Add(1)
		}
	})

	// the directory client or server used by the router needs the device TD
	deviceTDJson := td.MarshalTD(device1TD)
	err := dirSvc.CreateThing(deviceID, deviceTDJson)
	require.NoError(t, err)

	// to connect to the device, credentials are needed
	testAuthn.AddClient(testConsumerID, "", authn.ClientRoleOperator)
	token, _, err := testAuthn.CreateToken(testConsumerID, rpcTimeout)
	assert.NoError(t, err)
	routerMod.AddDeviceCredential(deviceID, clientID, token, td.SecSchemeBearer)

	// this should cause the router to connect to the device
	values, err := co.ReadAllProperties(deviceID)
	assert.NoError(t, err)
	assert.NotEmpty(t, values)

	// this should cause the router to connect to the device using the device TD
	err = co.ObserveProperty(deviceID, "")
	assert.NoError(t, err)

	co.WriteProperty(deviceID, testenv.CounterPropName, 33, true)
	time.Sleep(time.Millisecond)

	// expect at least 1 notification
	assert.Greater(t, notifCount.Load(), int32(0))

}

// Connect to a test device using the router with reconnect, and subscribe to events.
// in this setup the stand-alone device runs a server and the router lives on the consumer.
// This test forcefully disconnects the consumer and verifies it auto reconnects with
// subscription restored.
func TestSubscribeReconnectToDevice(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const deviceID = "device-1"
	const clientID = "router1"
	event1Name := "event1"
	event1Value := "value1"
	event2Value := "value2"
	prop1Name := "prop1"
	prop1Value1 := "prop1value1"
	prop1Value2 := "prop1value2"
	var rxValue = new(atomic.Value)
	// rxChan := make(chan *msg.NotificationMessage, 1)

	// 1. Setup the test device with server and a TD
	// The test device runs a server. The router will have to match its security as
	// included in its TD
	testDevice, tdoc, testServer, stopFn := startTestServerDevice(deviceID)
	defer stopFn()

	// 2. setup the consumer side: directory, router and consumer
	// register the device TD in the directory for use by the router
	// See also the factory consumer recipes for this use-case that makes it easier.
	var testDirMod = directorypkg.NewDirectoryService("", "", nil, nil)
	err := testDirMod.Start()
	require.NoError(t, err)
	defer testDirMod.Stop()
	deviceTDJson := td.MarshalTD(tdoc)
	err = testDirMod.CreateThing(deviceID, deviceTDJson)
	require.NoError(t, err)

	// the router uses the TD to connect to the device.
	// this doesn't actually need a directory. GetTD could also simply return the device TD.
	routerMod := routerpkg.NewRouterService(storageDir, testDirMod.GetTD, nil, certsBundle.CaCert, rpcTimeout)
	err = routerMod.Start()
	require.NoError(t, err)
	defer routerMod.Stop()

	// to connect to the device, consumer credentials are needed
	testAuthn.AddClient(testConsumerID, "", authn.ClientRoleOperator)
	token, _, _ := testAuthn.CreateToken(testConsumerID, rpcTimeout)
	routerMod.AddDeviceCredential(deviceID, clientID, token, td.SecSchemeBearer)

	ctx, cancelFn := context.WithTimeout(context.Background(), rpcTimeout)

	// a consumer links to the router which connects to devices using device TDs
	co := consumer.NewConsumer(routerMod, func(notif *msg.NotificationMessage) {
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
	err = co.Start()
	assert.NoError(t, err)
	// this should cause the router to connect to the device using the device TD
	err = co.Subscribe(deviceID, "")
	assert.NoError(t, err)

	// 3. the device updates a property and event which should be received.
	testDevice.ExposedThing.PubProperty(deviceID, prop1Name, prop1Value1, false)
	testDevice.ExposedThing.PubEvent(deviceID, event1Name, event1Value)
	<-ctx.Done()
	assert.Equal(t, event1Value, rxValue.Load())

	//--- phase 2 force a disconnect

	// drop client connections
	slog.Info("--- breaking connections on the server, expect a warning ---")
	testServer.CloseAll()

	// reading properties should fail while auto-reconnect is ongoing
	slog.Info("---ReadAllProperties (while reconnecting)---")

	values, err := co.ReadAllProperties(deviceID)
	assert.Error(t, err)

	// lets sleep to allow for reconnect
	time.Sleep(time.Second)

	// publish a property should now succeed
	testDevice.ExposedThing.PubProperty(deviceID, prop1Name, prop1Value2, false)

	values, err = co.ReadAllProperties(deviceID)
	assert.NoError(t, err)

	time.Sleep(time.Millisecond) // time to receive
	assert.Equal(t, prop1Value2, values[prop1Name])

	// on reconnect, subscription should remain intact and event should be received
	testDevice.ExposedThing.PubEvent(deviceID, event1Name, event2Value)
	time.Sleep(time.Millisecond) // time to receive
	assert.Equal(t, event2Value, rxValue.Load())

}
