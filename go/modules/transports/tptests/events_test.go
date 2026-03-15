package tptests

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// test event messages between agent, server and client
// this uses the client and server helpers defined in connect_test.go

// Test subscribing and receiving all events by consumer
func TestSubscribeAll(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var rxVal atomic.Value
	var testMsg1 = "hello world 1"
	var testMsg2 = "hello world 2"
	var agentID = "agent1"
	var thingID = "dtw:thing1"
	var eventKey = "event11"
	var agentRxEvent atomic.Bool

	// 1. start the servers
	testEnv, cancelFn := StartTestEnv(defaultProtocol)
	defer cancelFn()

	// 2. connect as consumers
	co1, cc1, _ := testEnv.NewConsumerClient(testClientID1, authnapi.ClientRoleViewer, nil)
	defer cc1.Close()

	co2, cc2, _ := testEnv.NewConsumerClient(testClientID1, authnapi.ClientRoleViewer, nil)
	defer cc2.Close()

	// test agents are wired to be usable as a consumer
	// Note that those agents should have an apprequest handler set to avoid looping.
	agent1, agConn1, _ := testEnv.NewRCAgent(agentID, nil)
	defer agConn1.Close()

	// set the handler for events and subscribe
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFn()

	co1.SetNotificationSink(func(ev *msg.NotificationMessage) {
		slog.Info("client 1 receives event")
		// receive event
		rxVal.Store(ev.Data)
		//cancelFn()
	})
	co2.SetNotificationSink(func(ev *msg.NotificationMessage) {
		slog.Info("client 2 receives event")
	})
	agent1.SetNotificationHook(func(ev *msg.NotificationMessage) {
		// receive event, tests whether agents work as a consumer
		slog.Info("Agent receives event")
		agentRxEvent.Store(true)
		cancelFn()
	})

	// Subscribe to events. Each transport binding implements this as per its spec
	err := co1.Subscribe("", "")
	assert.NoError(t, err)
	err = co2.Subscribe(thingID, eventKey)
	assert.NoError(t, err)
	// agent1 acts as a consumer here, its must have its sink set to a client
	// so its requests can be forwarded.
	err = agent1.Subscribe("", "")
	assert.NoError(t, err)

	// 3. Server sends event to consumers
	time.Sleep(time.Millisecond * 10)
	notif1 := msg.NewNotificationMessage(
		agentID, msg.AffordanceTypeEvent, thingID, eventKey, testMsg1)
	testEnv.Server.SendNotification(notif1)

	// 4. subscriber should have received them
	<-ctx.Done()
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, testMsg1, rxVal.Load())
	assert.True(t, agentRxEvent.Load())

	// Unsubscribe from events
	err = co1.Unsubscribe("", "")
	assert.NoError(t, err)
	time.Sleep(time.Millisecond * 10) // async take time
	err = co2.Unsubscribe(thingID, eventKey)
	assert.NoError(t, err)
	err = agent1.Unsubscribe("", "")
	assert.NoError(t, err)
	agentRxEvent.Store(false)

	// 5. Server sends another event to consumers
	notif2 := msg.NewNotificationMessage(
		agentID, msg.AffordanceTypeEvent, thingID, eventKey, testMsg2)
	testEnv.Server.SendNotification(notif2)
	time.Sleep(time.Millisecond)
	// update not received
	assert.Equal(t, testMsg1, rxVal.Load(), "Unsubscribe didnt work")
	assert.False(t, agentRxEvent.Load())

	//
}

// Agent sends events to server
// This is the normal setup if the Thing agent is connected via a client using connection reversal
func TestPublishEventsByAgent(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var evVal atomic.Value
	var testMsg = "hello world"
	var thingID = "thing1"
	var eventKey = "event11"

	// 1. start the transport
	// handler of event notification on the server
	notificationHandler := func(msg *msg.NotificationMessage) {
		// the server handler receives all notifications
		if msg.ThingID == thingID {
			evVal.Store(msg.Data)
		}
	}
	testEnv, cancelFn := StartTestEnv(defaultProtocol)
	testEnv.Server.SetNotificationSink(notificationHandler)
	defer cancelFn()

	// 2. connect an agent to the server - eg connection reversal
	agent1, agConn1, _ := testEnv.NewRCAgent(testAgentID1, nil)
	defer agConn1.Close()

	// 3. agent publishes an event
	//  the agent is the sink of the client. Client connection will send notifications to server.
	agent1.PubEvent(thingID, eventKey, testMsg)
	time.Sleep(time.Millisecond) // time to take effect

	// event received by server
	rxMsg2 := evVal.Load()
	require.NotNil(t, rxMsg2)
	assert.Equal(t, testMsg, rxMsg2)
}

// Consumer reads events from agent
func TestReadEvent(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var thingID = "thing1"
	var eventKey = "event11"
	var eventValue = "value11"
	var timestamp = "eventtime"

	// 1. start the agent transport with the request handler
	// in this case the consumer connects to the agent (unlike when using a hub)
	agentReqHandler := func(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
		if req.Operation == wot.HTOpReadEvent && req.ThingID == thingID && req.Name == eventKey {
			evNotif := msg.NewNotificationMessage("agent1", msg.AffordanceTypeEvent, thingID, req.Name, eventValue)
			evNotif.Timestamp = timestamp

			resp := req.CreateResponse(evNotif, nil)
			resp.Timestamp = timestamp
			return replyTo(resp)
		}
		resp := req.CreateResponse(nil, errors.New("unexpected request"))
		return replyTo(resp)
	}

	testEnv, cancelFn := StartTestEnv(defaultProtocol)
	testEnv.Server.SetRequestSink(agentReqHandler)
	defer cancelFn()

	// 2. connect as a consumer
	co1, cc1, _ := testEnv.NewConsumerClient(testClientID1, authnapi.ClientRoleViewer, nil)
	defer cc1.Close()

	evNotif, err := co1.ReadEvent(thingID, eventKey)
	require.NoError(t, err)
	require.NotEmpty(t, evNotif)
	assert.Equal(t, eventValue, evNotif.Data)

}

// Consumer reads events from agent
//func TestReadAllEvents(t *testing.T) {
//	t.Logf("---%s---\n", t.Name())
//	var thingID = "thing1"
//	var event1Name = "event1"
//	var event2Name = "event2"
//	var event1Value = "value1"
//	var event2Value = "value2"
//
//	// 1. start the agent transport with the request handler
//	// in this case the consumer connects to the agent (unlike when using a hub)
//	agentReqHandler := func(req *transports.RequestMessage, c transports.IConnection) *transports.ResponseMessage {
//		if req.Operation == wot.HTOpReadAllEvents {
//			output := make(map[string]*transports.ResponseMessage)
//			output[event1Name] = transports.NewResponseMessage(wot.OpSubscribeEvent, thingID, event1Name, event1Value, nil, "")
//			output[event2Name] = transports.NewResponseMessage(wot.OpSubscribeEvent, thingID, event2Name, event2Value, nil, "")
//			resp := req.CreateResponse(output, nil)
//			return resp
//		}
//		return req.CreateResponse(nil, errors.New("unexpected request"))
//	}
//	srv, cancelFn := StartTransportServer(agentReqHandler, nil)
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
