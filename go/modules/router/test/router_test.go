package router_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"
	certstest "github.com/hiveot/hivekit/go/modules/certs/test"
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/router"
	httpserverapi "github.com/hiveot/hivekit/go/modules/transports/httpserver/api"
	"github.com/hiveot/hivekit/go/modules/transports/tptests"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/vocab"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var storageDir = path.Join(os.TempDir(), "router-test")

var testDevicePort = 9993
var testDeviceAddress = fmt.Sprintf(":%d", testDevicePort)
var certsBundle = certstest.CreateTestCertBundle(utils.KeyTypeED25519)
var testAuthn = tptests.NewTestAuthenticator()

const testClientID = "client1"
const testClientPass = "client1pass"
const testClientToken = "client1Token"

// the test directory that holds this td. http server is not needed

// create a virtual device
func startTestDevice(agentID string, thingID string) (v *tptests.TestDevice) {

	testAuthn.AddClient(testClientID, "", authnapi.ClientRoleManager)
	testAuthn.SetPassword(testClientID, testClientPass)

	// create a test device with server
	cfg := httpserverapi.NewConfig(
		certsBundle.ServerAddr, testDevicePort,
		certsBundle.ServerCert, certsBundle.CaCert, testAuthn)

	var testTM *td.TD = td.NewTD(thingID, "test device", vocab.ThingDevice)

	v = tptests.NewTestDevice(cfg, agentID, testTM)
	err := v.Start("")
	if err != nil {
		panic("failed starting test device")
	}
	return v
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

	var testDirMod = directory.NewDirectoryModule("", nil)
	err := testDirMod.Start("")
	require.NoError(t, err)
	m := router.NewRouterModule(storageDir, testDirMod.GetTD, nil, certsBundle.CaCert)
	err = m.Start("")
	require.NoError(t, err)
	defer m.Stop()
}

// connect to a virtual device and subscribe to events
func TestSubscribeToDevice(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const thingID1 = "thing-1"
	const agentID = "agent-1"
	const event1Name = "event1"
	const clientID = "client1"
	var event1Value string = "value1"
	var rxValue string

	// Setup the test device with server and a TD
	// FIXME: use the test server for auth
	testDevice := startTestDevice(agentID, thingID1)
	defer testDevice.Stop()
	req := msg.NewRequestMessage(wot.OpObserveAllProperties, thingID1, "", nil, "")
	testDevice.HandleRequest(req, func(resp *msg.ResponseMessage) error {
		return nil
	})
	deviceTDJson, _ := td.MarshalTD(testDevice.GetTD())

	// setup the consumer side: directory, router and consumer
	// register the device TD in the directory for use by the router
	var testDirMod = directory.NewDirectoryModule("", nil)
	err := testDirMod.Start("")
	require.NoError(t, err)
	err = testDirMod.CreateThing(agentID, deviceTDJson)
	require.NoError(t, err)

	// the router uses the TD to connect to the device.
	// this doesn't actually need a directory. GetTD could also simply return the device TD.
	routerMod := router.NewRouterModule(
		storageDir, testDirMod.GetTD, nil, certsBundle.CaCert)
	err = routerMod.Start("")
	require.NoError(t, err)
	defer routerMod.Stop()
	// to connect to the device, credentials are needed
	token, _, _ := testAuthn.CreateToken(testClientID, time.Minute)
	routerMod.AddThingCredential(thingID1, clientID, token)

	// a consumer links to the router and subscribes to the device
	// note for the purpose of this test the router can run on the client
	consumer := clients.NewConsumer("")
	consumer.SetRequestSink(routerMod.HandleRequest)
	routerMod.SetNotificationSink(consumer.HandleNotification)
	err = consumer.Start("")
	assert.NoError(t, err)
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
