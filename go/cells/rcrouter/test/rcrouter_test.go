package rcrouter_test

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/cells/authn"
	directory_service "github.com/hiveot/hivekit/go/cells/directory/service"
	rcrouter_service "github.com/hiveot/hivekit/go/cells/rcrouter/service"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// const rpcTimeout = time.Minute * 3 // allow for debugging breakpoints
const testConsumerID = "router1"

var testProtocol = api.HiveotGrpcTcpProtocolType

var testProtocols = []string{
	api.HiveotSseScProtocolType,
	api.HiveotGrpcTcpProtocolType,
	api.HiveotGrpcUnixProtocolType,
	api.HiveotWebsocketProtocolType,
	api.WotWebsocketProtocolType,
	// api.HttpBasicProtocolType,  // can't subscribe
}

// Start the server linked to a directory and a rc-router
func startService() (testEnv *testenv.TestEnv, stopFn func()) {

	testEnv = testenv.NewTestEnv(true)

	// 1. Start the server to use for the test protocol
	slog.Info("startTestServerDevice", "serverType", testProtocol)
	transportServer := testEnv.StartTestServer(testProtocol)

	testDirSvc, _ := directory_service.NewDirectoryService("", "", nil)
	testDirSvc.SetTimeout(testEnv.Env.RpcTimeout)

	rcrouter, _ := rcrouter_service.NewRCRouterService(testDirSvc.GetTD,
		func() []api.ITransportServer {
			return []api.ITransportServer{transportServer}
		})
	rcrouter.SetTimeout(testEnv.Env.RpcTimeout)

	// chain server->director->rcrouter
	transportServer.SetRequestSink(testDirSvc)
	testDirSvc.SetNotificationSink(transportServer)
	testDirSvc.SetRequestSink(rcrouter)
	rcrouter.SetNotificationSink(testDirSvc)

	testDirSvc.Start()
	rcrouter.Start()

	return testEnv, func() {
		testEnv.Stop()
		testDirSvc.Stop()
		rcrouter.Stop()
	}
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

// func TestConnectAllProtocols(t *testing.T) {
// 	for _, testProtocol = range testProtocols {
// 		t.Run("TestStartStop: "+testProtocol, TestStartStop)
// 		t.Run("TestReadObserveDeviceProperties: "+testProtocol, TestReadObserveDeviceProperties)
// 		t.Run("TestSubscribeReconnectToDevice: "+testProtocol, TestSubscribeReconnectToDevice)
// 	}
// }

// Generic directory store testcases
func TestStartStop(t *testing.T) {
	slog.Warn(fmt.Sprintf("---Test: %s %s---\n", t.Name(), testProtocol))
	const clientID = "testclient"

	testEnv, stopFn := startService()
	_ = testEnv
	defer stopFn()
}

// connect to a rc connected test device and subscribe to property updates
func TestReadRCDeviceProperties(t *testing.T) {
	slog.Warn(fmt.Sprintf("---Test: %s %s---\n", t.Name(), testProtocol))
	const deviceID = "device1"
	const consumerID = "user1"
	const prop1Name = "prop1"
	const prop1Value = "value1"
	var eThingID string

	// 1. Setup the test device with server and a TD
	// The test device connects to the server and publishes its TD.
	// For this test a consumer links directly to the router to send the read request.
	testEnv, stopFn := startService()
	defer stopFn()

	// 2. connect a device to the server - eg connection reversal - and publish a TD
	ething, deviceConn1, _ := testEnv.NewRCThing(deviceID, nil)
	eThingID = ething.GetThingID()
	defer deviceConn1.Close()
	tdoc := td.NewTD(eThingID, "test device", vocab.DeviceSensor)
	tdoc.AddProperty("", prop1Name, td.DataTypeString, prop1Value)
	// when the device publishes an observable property it becomes available for querying
	ething.PubProperty(eThingID, prop1Name, prop1Value, false)

	// publish the device TD to update the directory on the server.
	// This writes the TD over the RC-connection.

	// FIXME: who sets the senderID
	err := ething.PublishTD(tdoc.ToString())
	require.NoError(t, err)

	// 3. connect a consumer
	co1, cc1, _ := testEnv.NewTestConsumer(consumerID, authn.ClientRoleViewer)
	err = cc1.Connect()
	require.NoError(t, err)
	defer cc1.Close()

	// 4. Read the properties from the RC device
	values, err := co1.ReadAllProperties(eThingID)
	assert.NoError(t, err)
	require.Equal(t, 1, len(values))
	require.Equal(t, prop1Value, values[prop1Name])

	// todo: handle writeproperty in exposed thing
	// co1.WriteProperty(eThingID, testenv.CounterPropName, 33, true)
	// time.Sleep(time.Millisecond)

}
