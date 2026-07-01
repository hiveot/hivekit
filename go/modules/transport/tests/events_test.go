package transporttests

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/modules/thing"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllEventsProtocols(t *testing.T) {
	for _, testProtocol = range testProtocols {
		t.Run(testProtocol, TestAllEvents)
	}
}
func TestAllEvents(t *testing.T) {
	t.Run("TestSubscribeAll", TestSubscribeAll)
	t.Run("TestPublishEventsByThing", TestPublishEventsByRCThing)
	t.Run("TestReadEvent", TestReadEvent)
}

// test event messages between thing, server and client
// this uses the client and server helpers defined in connect_test.go
// Test subscribing and receiving all events by consumer
func TestSubscribeAll(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var rxVal atomic.Value
	var testMsg1 = "hello world 1"
	var testMsg2 = "hello world 2"
	var deviceID = "device1"
	var thingID = "thing1"
	var eventKey = "event11"

	// 1. start the servers
	testEnv, cancelFn := testenv.StartTestEnv(testProtocol, true)
	defer cancelFn()

	// 2. connect as consumers
	co1, cc1, _ := testEnv.NewConnectedConsumer(testClientID1, authn.ClientRoleViewer, false)
	defer cc1.Close()

	co2, cc2, _ := testEnv.NewConnectedConsumer(testClientID1, authn.ClientRoleViewer, false)
	defer cc2.Close()

	// set the handler for events and subscribe
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFn()

	co1.SetNotificationHook(func(ev *msg.NotificationMessage) {
		slog.Info("client 1 receives event")
		if ev.ThingID == thingID {
			// receive event, expect data from device
			rxVal.Store(ev.Data)
		}
	})
	co2.SetNotificationHook(func(ev *msg.NotificationMessage) {
		slog.Info("client 2 receives event")
		cancelFn()
	})

	// Subscribe to events. Each transport binding implements this as per its spec
	err := co1.Subscribe("", "")
	assert.NoError(t, err)
	err = co2.Subscribe(thingID, eventKey)
	assert.NoError(t, err)

	// 3. Server sends event to consumers
	time.Sleep(time.Millisecond * 10)
	notif1 := msg.NewNotificationMessage(
		deviceID, msg.AffordanceTypeEvent, thingID, eventKey, testMsg1)
	testEnv.Server.SendNotification(notif1)

	// 4. subscriber should have received them
	<-ctx.Done()
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, testMsg1, rxVal.Load())

	// Unsubscribe from events
	err = co1.Unsubscribe("", "")
	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 10) // async take time
	err = co2.Unsubscribe(thingID, eventKey)
	assert.NoError(t, err)

	// 5. Server sends another event to consumers
	notif2 := msg.NewNotificationMessage(
		deviceID, msg.AffordanceTypeEvent, thingID, eventKey, testMsg2)
	testEnv.Server.SendNotification(notif2)
	time.Sleep(time.Millisecond)
	// update not received
	assert.Equal(t, testMsg1, rxVal.Load(), "Unsubscribe didnt work")
}

// test if subscriptions are retained after a reconnect
func TestSubscribeReconnect(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const deviceID = "deviceID"
	var thingID = "thing1"
	var eventKey = "event11"
	var testMsg1 = "hello world 1"
	var notifEvent atomic.Int32
	var connectedCh = make(chan bool, 1)

	// 1. start the servers
	testEnv, cancelFn := testenv.StartTestEnv(testProtocol, true)
	defer cancelFn()

	// 2. connect a consumer with reconnect capability
	co1, cc1, _ := testEnv.NewConnectedConsumer(testClientID1, authn.ClientRoleViewer, true)
	defer cc1.Close()

	// Consumer subscribes to events.
	err := co1.Subscribe("", "")
	assert.NoError(t, err)
	co1.SetNotificationHook(func(notif *msg.NotificationMessage) {
		// receive event, tests whether devices work as a consumer
		slog.Info("consumer receives event",
			"name", notif.Name, "data", notif.ToString(0))
		notifEvent.Add(1)
		if notif.Name == api.ClientConnectionStatusEvent &&
			notif.Data.(api.ConnectionStatus) == api.StatusConnected {
			connectedCh <- true
		}
	})
	// 3. Server sends event to consumers
	time.Sleep(time.Millisecond * 10)
	notif1 := msg.NewNotificationMessage(
		deviceID, msg.AffordanceTypeEvent, thingID, eventKey, testMsg1)
	testEnv.Server.SendNotification(notif1)

	time.Sleep(time.Millisecond)

	// 4. Client should have received the event
	require.Equal(t, 1, int(notifEvent.Load()))

	// 5. close client connections and wait for reconnect
	slog.Warn("--- 1. disconnecting all clients---")
	notifEvent.Store(0)
	testEnv.Server.CloseAll()
	slog.Info("--- 2. Closed. start reconnecting")

	<-connectedCh
	slog.Info("--- 3. Reconnected, start resubscribing")

	// time to resubscribe
	time.Sleep(time.Millisecond * 10)

	// 6. send a new event and expect the consumer to receive it
	slog.Info("--- 4. sending notification", "name", notif1.Name)
	testEnv.Server.SendNotification(notif1)
	time.Sleep(time.Millisecond * 10)

	slog.Info("--- 5. check result", "nr notifications", notifEvent.Load())

	// 7. Client should have received the lost,connecting,  connected and notif1 events
	assert.Equal(t, 4, int(notifEvent.Load()))

}

// Device sends events to server using reverse connection.
func TestPublishEventsByRCThing(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var evVal atomic.Value
	var testMsg = "hello world"
	var thingID = "thing1"
	var eventKey = "event11"

	// 1. start the transport
	// handler of notifications received on the server
	co := consumer.NewConsumer(nil, func(msg *msg.NotificationMessage) {
		// the server handler receives all notifications
		if msg.ThingID == thingID {
			evVal.Store(msg.Data)
		}
	})
	testEnv, cancelFn := testenv.StartTestEnv(testProtocol, true)
	testEnv.Server.SetNotificationSink(co)
	defer cancelFn()

	// 2. connect a device to the server - eg connection reversal
	device1, deviceConn1, _ := testEnv.NewRCThing(testDeviceID1, nil)
	defer deviceConn1.Close()

	// 3. device publishes an event
	//  the device is the sink of the client. Client connection will send notifications to server.
	device1.PubEvent(thingID, eventKey, testMsg)
	time.Sleep(time.Millisecond) // time to take effect

	// event received by server
	rxMsg2 := evVal.Load()
	require.NotNil(t, rxMsg2)
	assert.Equal(t, testMsg, rxMsg2)
}

// Consumer reads events from device
func TestReadEvent(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var thingID = "thing1"
	var eventKey = "event11"
	var eventValue = "value11"
	var timestamp = "eventtime"

	// 1. start the device transport with the request handler
	// in this case the consumer connects to the device (unlike when using a hub)
	ag := thing.NewExposedThing("", func(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
		if req.Operation == td.HTOpReadEvent && req.ThingID == thingID && req.Name == eventKey {
			evNotif := msg.NewNotificationMessage("device1", msg.AffordanceTypeEvent, thingID, req.Name, eventValue)
			evNotif.Timestamp = timestamp

			resp := req.CreateResponse(evNotif, nil)
			resp.Timestamp = timestamp
			return replyTo(resp)
		}
		resp := req.CreateResponse(nil, errors.New("unexpected request"))
		return replyTo(resp)
	})

	testEnv, cancelFn := testenv.StartTestEnv(testProtocol, true)
	testEnv.Server.SetRequestSink(ag)
	defer cancelFn()

	// 2. connect as a consumer
	co1, cc1, _ := testEnv.NewConnectedConsumer(testClientID1, authn.ClientRoleViewer, false)
	defer cc1.Close()

	evNotif, err := co1.ReadEvent(thingID, eventKey)
	require.NoError(t, err)
	require.NotEmpty(t, evNotif)
	assert.Equal(t, eventValue, evNotif.Data)

}

// Consumer reads events from a device
//func TestReadAllEvents(t *testing.T) {
//	t.Logf("---%s---\n", t.Name())
//	var thingID = "thing1"
//	var event1Name = "event1"
//	var event2Name = "event2"
//	var event1Value = "value1"
//	var event2Value = "value2"
//
//	// 1. start the device transport with the request handler
//	// in this case the consumer connects to the device (unlike when using a hub)
//	reqHandler := func(req *transport.RequestMessage, c api.IConnection) *transport.ResponseMessage {
//		if req.Operation == td.HTOpReadAllEvents {
//			output := make(map[string]*transport.ResponseMessage)
//			output[event1Name] = transport.NewResponseMessage(td.OpSubscribeEvent, thingID, event1Name, event1Value, nil, "")
//			output[event2Name] = transport.NewResponseMessage(td.OpSubscribeEvent, thingID, event2Name, event2Value, nil, "")
//			resp := req.CreateResponse(output, nil)
//			return resp
//		}
//		return req.CreateResponse(nil, errors.New("unexpected request"))
//	}
//	srv, cancelFn := StartTransportServer(reqHandler, nil)
//	_ = srv
//	defer cancelFn()
//
//	// 2. connect as a consumer
//	cc1, consumer1, _ := NewConsumer(testClientID1, srv.GetForm)
//	defer cc1.Disconnect()
//
//	evMap, err := consumer1.ReadAllEvents(thingID)
//	require.NoError(t, err)
//	require.Equal(t, 2, len(evMap))
//	require.Equal(t, event1Value, evMap[event1Name].Value)
//	require.Equal(t, event2Value, evMap[event2Name].Value)
//}
