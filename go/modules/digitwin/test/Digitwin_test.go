package digitwin_test

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"
	"github.com/hiveot/hivekit/go/modules/digitwin"
	digitwinapi "github.com/hiveot/hivekit/go/modules/digitwin/api"
	"github.com/hiveot/hivekit/go/modules/digitwin/internal/module"
	"github.com/hiveot/hivekit/go/modules/directory"
	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
	directoryclient "github.com/hiveot/hivekit/go/modules/directory/client"
	"github.com/hiveot/hivekit/go/modules/router"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/tptests"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var storageRoot = ""

const rpcTimout = transports.DefaultRpcTimeout

// TestMain setup logging and creates a test environment
func TestMain(m *testing.M) {
	utils.SetLogging("info", "")

	result := m.Run()
	if result != 0 {
		println("Test failed with code:", result)
	} else {
	}

	os.Exit(result)
}

// startService initializes a service and a client
// This sets-up a module chain with a server, directory, digitwin, vcache, and router
func startService() (
	testEnv *tptests.TestEnv,
	dir directoryapi.IDirectoryServer,
	dtw digitwinapi.IDigitwinModule,
	stopFn func()) {

	// testEnv,cancelFn = tptests.StartTestEnv(transports.ProtocolSchemeWotWSS)
	testEnv = tptests.NewTestEnv()
	// http server needed for all communications
	testEnv.StartHttpServer()
	// a websocket server for RRN messaging
	appServer := testEnv.StartTestServer(transports.WotWebsocketProtocolType)

	// the directory server that will contain digitwin Things
	dir = directory.NewDirectoryModule("", testEnv.HttpServer)
	err := dir.Start("")
	if err != nil {
		panic("Failed to start directory server")
	}
	// the digitwin module to test, it will create its own vcache module
	dtw = digitwin.NewDigitwinModule(storageRoot, dir, appServer.AddTDSecForms)
	err = dtw.Start("")
	if err != nil {
		panic("unable to start the digitwin service")
	}
	// the router module uses the digitwin Thing Directory
	// getDeviceTD := dtw.GetDeviceDirectory().GetTD
	routerStorage := path.Join(os.TempDir(), "router-test")
	rtr := router.NewRouterModule(routerStorage,
		dtw.GetDeviceTD, []transports.ITransportServer{appServer}, testEnv.CertBundle.CaCert)
	rtr.SetTimeout(rpcTimout)

	// create a request pipeline server->directory->digitwin->router->server
	appServer.SetRequestSink(dir.HandleRequest)
	dir.SetRequestSink(dtw.HandleRequest)
	dtw.SetRequestSink(rtr.HandleRequest)
	rtr.SetRequestSink(appServer.HandleRequest)

	// create a notification pipeline server->router->digitwin->directory->server
	appServer.SetNotificationSink(rtr.HandleNotification)
	rtr.SetNotificationSink(dtw.HandleNotification)
	dtw.SetNotificationSink(dir.HandleNotification)
	dir.SetNotificationSink(appServer.HandleNotification)

	slog.Info("--- digitwin test environment started ---")

	return testEnv, dir, dtw, func() {
		// give client connections time to close
		time.Sleep(time.Millisecond)
		slog.Info("--- digitwin test environment stopping ---")
		dir.Stop()
		dtw.Stop()
		rtr.Stop()
		appServer.Stop()
		testEnv.HttpServer.Stop()
	}
}
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	testEnv, dir, dtw, stopFn := startService()
	defer stopFn()
	_ = testEnv
	_ = dir
	_ = dtw
}

// Write a TD to the directory and verify a digital twin is created using the module API
func TestCreateDigitwinTD(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"

	testEnv, dir, dtw, stopFn := startService()
	defer stopFn()

	// pretent to be an agent that writes a TD to the directory
	td1 := testEnv.CreateTestTD(0)
	td1Json, _ := td.MarshalTD(td1)
	err := dir.UpdateThing(agentID, td1Json)

	// 1. Retrieve the TD from the directory.
	dtwList, err := dir.RetrieveAllThings(0, 0)
	require.NoError(t, err)
	require.Equal(t, 1, len(dtwList))
	dtw1, err := td.UnmarshalTD(dtwList[0])
	require.NoError(t, err)

	// 2. The digitwin ID should have the dtw prefix
	assert.True(t, strings.HasPrefix(dtw1.ID, digitwinapi.DigitwinIDPrefix))

	// 3. check if properties, events and actions are still there
	require.Less(t, len(td1.Properties), len(dtw1.Properties)) // digitwin added properties
	require.Equal(t, len(td1.Events), len(dtw1.Events))
	require.Equal(t, len(td1.Actions), len(dtw1.Actions))

	// 4. check if the base form points to the server
	require.NotEmpty(t, dtw1.Base, "Missing base in TD")
	expectedBase := testEnv.Server.GetConnectURL()
	expectedProtocolType := testEnv.Server.GetProtocolType()
	assert.NotEmpty(t, expectedProtocolType)
	assert.Equal(t, expectedBase, dtw1.Base)

	// 5. check if the forms in the affordances are replaced
	for _, aff := range dtw1.Properties {
		require.NotEmpty(t, aff.Forms)
		form0 := aff.Forms[0]
		assert.NotEmpty(t, form0.GetOperations())
		subprotocol, _ := form0.GetSubprotocol()
		assert.Equal(t, subprotocol, transports.WotWebsocketSubprotocol)
	}
	for _, aff := range dtw1.Events {
		require.NotEmpty(t, aff.Forms)
		assert.NotEmpty(t, aff.Forms[0].GetOperations())
	}
	for _, aff := range dtw1.Actions {
		require.NotEmpty(t, aff.Forms)
		assert.NotEmpty(t, aff.Forms[0].GetOperations())
	}

	require.NoError(t, err)
	_ = dtw
}

// Read a property from the digital twin
func TestReadDigitwinProperty(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	const userID = "user1"
	const prop1Name = "prop-0" // generated by test env
	const prop1Value = "value1"
	var rxPropValue string

	testEnv, dir, dtw, stopFn := startService()
	_ = dtw
	defer stopFn()

	deviceTD1 := testEnv.CreateTestTD(0)

	// the digital twin sink will receive the request to read property
	// this tests if the dtw would forward it downstream as property is unknown.
	dtw.SetRequestSink(func(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
		if req.Operation == wot.OpReadProperty {
			if req.ThingID == deviceTD1.ID && req.Name == prop1Name {
				resp := req.CreateResponse(prop1Value, nil)
				err := replyTo(resp)
				return err
			}
		}
		return fmt.Errorf("unknown request ")
	})

	// 1: create a consumer that subscribes to notifications
	co, cc1, _ := testEnv.NewConsumerClient(userID, authnapi.ClientRoleViewer, nil)
	err := co.ObserveProperty("", prop1Name)
	require.NoError(t, err)
	defer cc1.Stop()
	// expect a digital twin notification from changing the device property
	dtwThingID := module.MakeDigitwinID(agentID, deviceTD1.ID)
	co.SetNotificationHook(func(notif *msg.NotificationMessage) {
		rxPropValue = notif.ToString(0)
		require.NotEmpty(t, rxPropValue)
		assert.Equal(t, dtwThingID, notif.ThingID)
		assert.Equal(t, prop1Name, notif.Name)
		slog.Info("*** Received notification",
			"type", notif.AffordanceType, "thingID", notif.ThingID, "name", notif.Name)
	})

	// 2. pretent to be an agent that writes a TD to the directory
	td1Json, _ := td.MarshalTD(deviceTD1)
	err = dir.UpdateThing(agentID, td1Json)
	assert.NoError(t, err)

	// 3. the agent emits a property notification from the thing to the server
	// in the test setup the directory is the first module in the pipeline
	ag, cc2, _ := testEnv.NewRCAgent(agentID, nil)
	defer cc2.Close()

	ag.PubProperty(deviceTD1.ID, prop1Name, prop1Value)
	// let the communication proceed
	time.Sleep(time.Millisecond)

	// The digital twin module receives this thing notification and updates the
	// digital twin property state.
	assert.Equal(t, prop1Value, rxPropValue)

	// 4. the consumer reads the property value
	var respValue string
	err = co.Rpc(wot.OpReadProperty, dtwThingID, prop1Name, nil, &respValue)
	require.NoError(t, err)
	assert.Equal(t, prop1Value, respValue)
}

// Write a property via the digital twin
func TestWriteDigitwinProperty(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	const userID = "user1"
	const prop1Name = "prop-0" // generated by test env
	const prop1Value = "value1"
	var txPropValue string

	testEnv, dir, dtw, stopFn := startService()
	_ = dtw
	_ = dir
	defer stopFn()

	// 1: create a consumer that writes a property
	co, cc1, _ := testEnv.NewConsumerClient(userID, authnapi.ClientRoleViewer, nil)
	err := co.ObserveProperty("", prop1Name)
	require.NoError(t, err)
	defer cc1.Stop()

	// 2. create an agent that receives the write request
	ag, cc2, _ := testEnv.NewRCAgent(agentID, nil)
	defer cc2.Close()
	ag.SetAppRequestHandler(func(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
		if req.Operation == wot.OpWriteProperty {
			txPropValue = req.ToString(0)
			resp := req.CreateResponse(nil, nil)

			// write property should send a notification that updates the digital twin
			go ag.PubProperty(req.ThingID, req.Name, txPropValue)

			return replyTo(resp)
		} else if req.ThingID == directoryapi.DefaultDirectoryServiceID {
			// this is a request for the directory. Forward it
			return ag.ForwardRequest(req, replyTo)
		} else {
			resp := req.CreateResponse(nil, fmt.Errorf("unexpected request"))
			return replyTo(resp)
		}
	})

	// 3. write a TD using the directory client
	// normally the discovery process discovered the directory service service ID,
	// but most likely it uses the default.
	td1 := testEnv.CreateTestTD(0)
	td1Json, _ := td.MarshalTD(td1)
	err = directoryclient.UpdateTD(
		directoryapi.DefaultDirectoryServiceID, td1Json, ag.ForwardRequest)

	assert.NoError(t, err)
	// check whether the td is now in the directory
	dtwThing1ID := module.MakeDigitwinID(agentID, td1.ID)
	td2Json, err := dir.RetrieveThing(dtwThing1ID)
	require.NoError(t, err)
	require.NotEmpty(t, td2Json)
	// check whether the agentID is set
	tdi2, err := td.UnmarshalTD(td2Json)
	require.NoError(t, err)
	assert.Equal(t, agentID, tdi2.AgentID)

	// 4. Consumer reads the TD with its own directory client
	dirCoCl := directoryclient.NewDirectoryMsgClient("", co)
	td3Json, err := dirCoCl.RetrieveThing(dtwThing1ID)
	require.NoError(t, err)
	td3, err := td.UnmarshalTD(td3Json)
	require.NoError(t, err)
	require.NotEmpty(t, td3)

	// 5. Consumer writes the property
	err = co.WriteProperty(dtwThing1ID, prop1Name, prop1Value, true)
	require.NoError(t, err)
	assert.Equal(t, prop1Value, txPropValue)
	// need some time for the agent notification to update the digital twin value
	time.Sleep(time.Millisecond * 10)

	// 6. Consumer reads the property
	rxPropValue, err := co.ReadProperty(dtwThing1ID, prop1Name)
	require.NoError(t, err)
	assert.Equal(t, prop1Value, rxPropValue)
}

// Write a property via the digital twin
func TestInvokeDigitwinAction(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	const userID = "user1"
	const actionName = "action-0" // generated by test env
	const actionInput = "value1"
	const thingID = "thing1"
	dtwThing1ID := module.MakeDigitwinID(agentID, thingID)

	testEnv, dir, dtw, stopFn := startService()
	_ = dtw
	_ = dir
	defer stopFn()

	// 1: create a consumer
	co, cc1, _ := testEnv.NewConsumerClient(userID, authnapi.ClientRoleViewer, nil)
	// the action will submit an event
	err := co.Subscribe(dtwThing1ID, actionName)
	require.NoError(t, err)
	defer cc1.Stop()

	// 2. create an RC agent that receives the action request
	ag, cc2, _ := testEnv.NewRCAgent(agentID, nil)
	defer cc2.Close()
	ag.SetAppRequestHandler(func(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
		// echo the input
		if req.Operation == wot.OpInvokeAction {
			actionInput := req.ToString(0)
			resp := req.CreateResponse(actionInput, nil)
			// submit an event after the action
			go ag.PubEvent(req.ThingID, req.Name, req.Input)
			return replyTo(resp)
		} else if req.ThingID == directoryapi.DefaultDirectoryServiceID {
			// this is a request for the directory. Forward it
			return ag.ForwardRequest(req, replyTo)
		} else {
			resp := req.CreateResponse(nil, fmt.Errorf("unexpected request"))
			return replyTo(resp)
		}
	})

	// 3. write a TD with this action
	td1 := testEnv.CreateTestTD(0)
	td1.ID = thingID
	td1Json, _ := td.MarshalTD(td1)
	err = directoryclient.UpdateTD(
		directoryapi.DefaultDirectoryServiceID, td1Json, ag.ForwardRequest)
	assert.NoError(t, err)

	// 4. Consumer invokes the first action
	// note that this doesnt use the TD. It should still work
	var respValue string
	err = co.InvokeAction(dtwThing1ID, actionName, actionInput, &respValue)
	require.NoError(t, err)
	// the handler above returns the input value
	assert.Equal(t, actionInput, respValue)
	// need some time for the event notification to update the digital twin value
	time.Sleep(time.Millisecond * 10)

	// 6. Consumer reads the event
	rxEvent, err := co.ReadEvent(dtwThing1ID, actionName)
	require.NoError(t, err)
	assert.Equal(t, actionInput, rxEvent.Data)
}
