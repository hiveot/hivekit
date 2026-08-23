package router_test

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/authn"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/cells/directory"
	directory_service "github.com/hiveot/hivekit/go/cells/directory/service"
	"github.com/hiveot/hivekit/go/cells/router"
	router_service "github.com/hiveot/hivekit/go/cells/router/service"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var storageDir = path.Join(os.TempDir(), "hivekit", "router-test")

// var testDevicePort = 9993
// var certsBundle = certstest.CreateTestCertBundle(utils.KeyTypeED25519)
// var testAuthn = testenv.NewTestAuthenticator()

const rpcTimeout = time.Minute * 3 // allow for debugging breakpoints
const testConsumerID = "router1"

var testProtocol = api.WotWebsocketProtocolType

var testProtocols = []string{
	api.HiveotSseScProtocolType,
	api.HiveotGrpcTcpProtocolType,
	api.HiveotGrpcUnixProtocolType,
	api.HiveotWebsocketProtocolType,
	api.WotWebsocketProtocolType,
	// api.HttpBasicProtocolType,  // can't subscribe
}

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
	tdoc *td.TD, testEnv *testenv.TestEnv, stopFn func()) {

	testEnv = testenv.NewTestEnv(true)

	// 1. Start the server to use for the test protocol
	slog.Info("startTestServerDevice", "deviceID", deviceID, "serverType", testProtocol)
	transportServer := testEnv.StartTestServer(testProtocol)

	// 2. Create the test device Thing and link it to the server so it receives requests
	testDevice = testenv.NewCounterDevice(deviceID, nil)
	testDevice.SetNotificationSink(transportServer)
	transportServer.SetRequestSink(testDevice)
	err := testDevice.Start()
	if err != nil {
		panic("startTestServerDevice: failed starting test device")
	}

	// Add the connection forms to the device TD
	tdJson := testDevice.GetTD()
	tdoc, _ = td.UnmarshalTD(tdJson)
	transportServer.AddTDSecForms(tdoc, false)

	// tdoc describes how the router can connect to the testDevice
	return testDevice, tdoc, testEnv, func() {
		testDevice.Stop()
		testEnv.Stop()
	}
}

// Setup a consumer that uses the router and directory to connect to devices
// The router has a credentials store for authentication
func SetupConsumerWithRouter(
	rootCAs *x509.CertPool) (
	co *consumer.Consumer,
	routerSvc router.IRouterService,
	dirSvc directory.IDirectoryService,
) {
	const clientID = "clientID" // fallback ID to connect as client

	// setup the consumer side: directory, router and consumer
	// register the device TD in the directory for use by the router
	dirSvc = directory_service.NewDirectoryService("", storageDir, nil, nil)
	err := dirSvc.Start()
	if err != nil {
		panic("SetupConsumerWithRouter: Directory.Start: " + err.Error())
	}

	// the router uses the TD to connect to the device.
	// this doesn't actually need a directory. GetTD could also simply return the device TD.
	routerSvc = router_service.NewRouterService(
		storageDir, false, clientID, nil, rootCAs, rpcTimeout, dirSvc.GetTD, nil)

	err = routerSvc.Start()
	if err != nil {
		panic("SetupConsumerWithRouter: Router.Start: " + err.Error())
	}

	// A consumer links to the router and subscribes to the device.
	// For the purpose of this test the router runs client side.
	co = consumer.NewConsumer(routerSvc, nil)
	co.SetTimeout(rpcTimeout)
	err = co.Start()
	if err != nil {
		panic("SetupConsumerWithRouter: Consumer.Start: " + err.Error())
	}
	return co, routerSvc, dirSvc
}

// TestMain create a test folder for certificates and private key
func TestMain(m *testing.M) {
	utils.SetLogging("info", "")

	os.RemoveAll(storageDir)

	result := m.Run()
	if result != 0 {
		println("Test failed with code:", result)
	} else {
	}

	os.Exit(result)
}

func TestConnectAllProtocols(t *testing.T) {
	for _, testProtocol = range testProtocols {
		t.Run("TestStartStop", TestStartStop)
		t.Run("TestReadObserveDeviceProperties", TestReadObserveDeviceProperties)
		t.Run("TestSubscribeReconnectToDevice", TestSubscribeReconnectToDevice)
	}
}

// Generic directory store testcases
func TestStartStop(t *testing.T) {
	slog.Warn(fmt.Sprintf("---Test: %s %s---\n", t.Name(), testProtocol))
	const clientID = "testclient"

	var testDirMod = directory_service.NewDirectoryService("", "", nil, nil)
	err := testDirMod.Start()
	require.NoError(t, err)
	// test no cred store
	m := router_service.NewRouterService(
		"", false, clientID, nil, nil, rpcTimeout, testDirMod.GetTD, nil)
	err = m.Start()
	require.NoError(t, err)
	defer m.Stop()
}

func TestCredentialsStore(t *testing.T) {
	slog.Warn(fmt.Sprintf("---Test: %s %s---\n", t.Name(), testProtocol))
	const thingID1 = "thing-1"
	const clientID = "client1"
	const clientCred = "secret"
	const thingScheme = td.SecSchemeBearer

	os.RemoveAll(storageDir)

	// the router uses the TD to connect to the device.
	// this doesn't actually need a directory. GetTD could also simply return the device TD.
	routerMod := router_service.NewRouterService(
		storageDir, false, clientID, nil, nil, rpcTimeout, nil, nil)
	err := routerMod.Start()
	require.NoError(t, err)

	credType, hasCred := routerMod.HasThingCredentials(thingID1)
	assert.False(t, hasCred)
	assert.Equal(t, "", credType)

	routerMod.AddDeviceCredential(thingID1, clientID, clientCred, thingScheme)

	credType, hasCred = routerMod.HasThingCredentials(thingID1)
	assert.True(t, hasCred)
	assert.Equal(t, thingScheme, credType)

	routerMod.Stop()

	// restarting the router should retain the credentials
	err = routerMod.Start()
	require.NoError(t, err)

	credType, hasCred = routerMod.HasThingCredentials(thingID1)
	assert.True(t, hasCred)
	routerMod.Stop()
}

// connect to a stand-alone test device and authenticate with client cert
func TestAuthClientCert(t *testing.T) {
	const deviceID = "device1"
	const clientID = "router1"
	const prop1Name = "prop1"
	const prop1Value = "value1"

	testDevice, device1TD, testEnv, stopFn := startTestServerDevice(deviceID)
	defer stopFn()
	// when the device publishes an observable property it becomes available for querying
	testDevice.ExposedThing.PubProperty(deviceID, prop1Name, prop1Value, false)

	// 2. setup the consumer with the router and directory client or service
	co, routerSvc, dirSvc := SetupConsumerWithRouter(testEnv.CertBundle.RootCAs)
	defer dirSvc.Stop()
	defer routerSvc.Stop()

	// 3. the directory (client or server) used by the router needs the device TD
	deviceTDJson := td.MarshalTD(device1TD)
	err := dirSvc.CreateThing(deviceID, deviceTDJson)
	require.NoError(t, err)

	// 4. use client cert as credentials to connect to the device
	testEnv.TestAuthn.AddClient(testConsumerID, "", authn.ClientRoleOperator)
	clientCert := testEnv.CertBundle.ClientCert
	routerSvc.SetClientCert(clientCert)
	// alt:
	// certPem, keyPem := utils.TLSCertToPEM(clientCert)
	// pemCred := certPem + "\n" + keyPem
	// routerMod.AddDeviceCredential(deviceID, clientID, pemCred, td.SecSchemeCert)

	// 5. Send a request, which causes the router to connect to the device
	values, err := co.ReadAllProperties(deviceID)
	assert.NoError(t, err)
	assert.NotEmpty(t, values)
}

// connect to a stand-alone test device and subscribe to property updates
func TestReadObserveDeviceProperties(t *testing.T) {
	slog.Warn(fmt.Sprintf("---Test: %s %s---\n", t.Name(), testProtocol))
	const deviceID = "device1"
	const clientID = "router1"
	const prop1Name = "prop1"
	const prop1Value = "value1"
	var notifCount atomic.Int32

	// 1. Setup the test device with server and a TD
	// The test device is a runs a server that passes requests to its device.
	// The router will have to match its security as described in the device TD
	// The device of the test device handles read requests
	testDevice, device1TD, testEnv, stopFn := startTestServerDevice(deviceID)
	defer stopFn()
	testAuthn := testEnv.TestAuthn

	// when the device publishes an observable property it becomes available for querying
	testDevice.ExposedThing.PubProperty(deviceID, prop1Name, prop1Value, false)

	// 2. setup the consumer with the router and directory client or service
	co, routerMod, dirSvc := SetupConsumerWithRouter(testEnv.CertBundle.RootCAs)
	defer dirSvc.Stop()
	defer routerMod.Stop()

	// test if subscribe works
	co.SetNotificationHook(func(notif *msg.NotificationMessage) {
		if notif.AffordanceType == msg.AffordanceTypeProperty {
			notifCount.Add(1)
		}
	})

	// 3. the directory (client or server) used by the router needs the device TD
	deviceTDJson := td.MarshalTD(device1TD)
	err := dirSvc.CreateThing(deviceID, deviceTDJson)
	require.NoError(t, err)

	// 4. to connect to the device, credentials are needed
	testAuthn.AddClient(testConsumerID, "", authn.ClientRoleOperator)
	token, _, err := testAuthn.CreateToken(testConsumerID, rpcTimeout)
	assert.NoError(t, err)
	routerMod.AddDeviceCredential(deviceID, clientID, token, td.SecSchemeBearer)

	// 5. Send a request, which causes the router to connect to the device
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
	slog.Warn(fmt.Sprintf("---Test: %s %s---\n", t.Name(), testProtocol))
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
	testDevice, tdoc, testEnv, stopFn := startTestServerDevice(deviceID)
	defer stopFn()

	// 2. setup the consumer side: directory, router and consumer
	// register the device TD in the directory for use by the router
	// See also the factory consumer recipes for this use-case that makes it easier.
	var testDirMod = directory_service.NewDirectoryService("", "", nil, nil)
	err := testDirMod.Start()
	require.NoError(t, err)
	defer testDirMod.Stop()
	deviceTDJson := td.MarshalTD(tdoc)
	err = testDirMod.CreateThing(deviceID, deviceTDJson)
	require.NoError(t, err)

	// the router uses the TD to connect to the device.
	// this doesn't actually need a directory. GetTD could also simply return the device TD.
	routerMod := router_service.NewRouterService(
		storageDir, true, clientID,
		testEnv.CertBundle.ClientCert,
		testEnv.CertBundle.RootCAs,
		rpcTimeout, testDirMod.GetTD, nil)
	err = routerMod.Start()
	require.NoError(t, err)
	defer routerMod.Stop()

	// to connect to the device, consumer credentials are needed
	testEnv.TestAuthn.AddClient(testConsumerID, "", authn.ClientRoleOperator)
	token, _, _ := testEnv.TestAuthn.CreateToken(testConsumerID, rpcTimeout)
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
	slog.Warn("--- breaking connections on the server, expect a warning ---")
	testEnv.Server.CloseAll()

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
