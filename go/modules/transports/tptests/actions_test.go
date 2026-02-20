package tptests

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teris-io/shortid"
)

// TestInvokeActionFromConsumerToServer: classic 'consumer talks to the server'
// as if it is a Thing. In this test the server replies.
// (routing is not part of this package)
func TestInvokeActionFromConsumerToServer(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	//var outputVal atomic.Value
	var testOutput string
	// var testActionStatus msg.ActionStatus

	var inputVal atomic.Value
	var testMsg1 = "hello world 1"
	var thingID = "thing1"
	var actionName = "action1"

	// the server will receive the action request and return an immediate result
	handleRequest := func(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
		var resp *msg.ResponseMessage
		if req.Operation == wot.OpInvokeAction {
			inputVal.Store(req.Input)
			// CreateActionResponse returns ActionStatus
			resp = req.CreateResponse(req.Input, nil)
		} else {
			assert.Fail(t, "Not expecting operation: "+req.Operation)
			resp = req.CreateResponse(nil, fmt.Errorf("Unexpected request operation '%s'", req.Operation))
		}
		return replyTo(resp)
	}
	// 1. start the servers
	testEnv, cancelFn := StartTestEnv(defaultProtocol)
	defer cancelFn()
	testEnv.Server.SetRequestSink(handleRequest)

	// 2. connect a client
	co1, cc1, token := testEnv.NewConsumerClient(testClientID1, authn.ClientRoleViewer, nil)
	defer cc1.Close()
	require.NotEmpty(t, token)
	ctx1, release1 := context.WithTimeout(context.Background(), time.Minute)
	defer release1()

	// the response handler
	responseHandler := func(resp *msg.ResponseMessage) error {
		slog.Info("testOutput was updated asynchronously via the message handler")

		// response should be an ActionStatus object
		err2 := utils.Decode(resp.Output, &testOutput)
		release1()
		return err2
	}

	// 3. invoke the action without waiting for a result
	// the response handler above will receive the result
	// testOutput can be updated as an immediate result or via the callback message handler
	req := msg.NewRequestMessage(wot.OpInvokeAction, thingID, actionName, testMsg1, shortid.MustGenerate())
	err := co1.SendRequest(req, responseHandler)

	require.NoError(t, err)
	<-ctx1.Done()

	// whether receiving completed or delivered depends on the binding
	require.Equal(t, testMsg1, testOutput)

	// 4. verify that the server received it and send a reply
	assert.NoError(t, err)
	assert.Equal(t, testMsg1, inputVal.Load())
	assert.Equal(t, testMsg1, testOutput)

	// 5. Again but wait for the action result
	var result1 string
	err = co1.InvokeAction(thingID, actionName, testMsg1, &result1)
	assert.NoError(t, err)
	assert.Equal(t, testMsg1, result1)
}

// Warning: this is a bit of a mind bender if you're used to classic consumer->thing interaction.
// This test uses a Thing agent as a client and have it reply to a request from the server.
// The server in this case passes on a message received from a consumer, which is also a client.
// This reflects the use-case of agents using connection reversal to gateways.
func TestInvokeActionFromServerToAgent(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var reqVal atomic.Value
	var replyVal atomic.Value
	var testMsg1 = "hello world 1"
	var testMsg2 = "hello world 2"
	var thingID = "thing1"
	var actionKey = "action1"
	var corrID = "correlation-1"

	// 1. start the server. register a message handler for receiving an action status
	// async reply from the agent after the server sends an invoke action.
	// Note that WoT doesn't cover this use-case so this uses hiveot vocabulary operation.

	ctx1, cancelFn1 := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancelFn1()
	// server receives agent response
	responseHandler := func(resp *msg.ResponseMessage) error {
		var responseData string
		// The server receives a response message from the agent
		// (which normally is forwarded to the remote consumer; but not in this test)
		assert.NotEmpty(t, resp.CorrelationID)
		assert.Equal(t, wot.OpInvokeAction, resp.Operation)

		slog.Info("Server: received response from agent",
			"op", resp.Operation,
			"output", resp.Output,
		)
		err := resp.Decode(&responseData)
		assert.NoError(t, err)

		replyVal.Store(responseData)
		cancelFn1()
		return nil
	}
	// tmpSink := &modules.HiveModuleBase{}
	// tmpSink.SetResponseHandler(responseHandler)
	testEnv, cancelFn2 := StartTestEnv(defaultProtocol)
	defer cancelFn2()

	// 2a. connect as an agent
	ag1client, cc1, token := testEnv.NewRCAgent(testAgentID1)
	require.NotEmpty(t, token)
	defer cc1.Close()

	// an agent receives requests from the server
	ag1client.SetAppRequestHandler(func(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
		// agent receives action request and returns a result
		slog.Info("Agent receives request", "op", req.Operation)
		assert.Equal(t, testClientID1, req.SenderID)
		reqVal.Store(req.Input)
		go func() {
			time.Sleep(time.Millisecond)
			// separately send a completed response
			resp := req.CreateResponse(testMsg2, nil)
			slog.Info("Agent sends response", "op", req.Operation)
			err2 := replyTo(resp)
			assert.NoError(t, err2)
		}()
		// the response is sent asynchronously
		return nil
	})

	// Send the action request from the server to the agent (the agent is connected as a client)
	// and expect result using the request status message sent by the agent.
	time.Sleep(time.Millisecond)
	// ag1Server := srv.GetConnectionByClientID(testAgentID1)
	// require.NotNil(t, ag1Server)

	req := msg.NewRequestMessage(wot.OpInvokeAction, thingID, actionKey, testMsg1, corrID)
	req.SenderID = testClientID1
	req.CorrelationID = "rpc-TestInvokeActionFromServerToAgent"
	// err := ag1Server.SendRequest(req)
	err := testEnv.Server.SendRequest(testAgentID1, req, responseHandler)
	require.NoError(t, err)

	// wait until the agent has sent a reply
	<-ctx1.Done()
	time.Sleep(time.Millisecond * 10)

	// if all went well the agent received the request and the server its response
	assert.Equal(t, testMsg1, reqVal.Load())
	assert.Equal(t, testMsg2, replyVal.Load())
}

// TestQueryActions consumer queries the server for actions
// The server receives a QueryAction request and sends a response
func TestQueryActions(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var testMsg1 = "hello world 1"
	var thingID = "dtw:thing1"
	var actionKey = "action1"

	// 1. start the server. register a request handler for receiving a request
	// from the agent after the server sends an invoke action.
	// Note that WoT doesn't cover this use-case so this uses hiveot vocabulary operation.
	requestHandler := func(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
		var resp *msg.ResponseMessage
		assert.NotNil(t, replyTo)
		assert.NotNil(t, req.CorrelationID)
		switch req.Operation {
		case wot.OpQueryAction:
			// reply a response carrying the queried action status
			actStat := &msg.ResponseMessage{
				ThingID:   req.ThingID,
				Name:      req.Name,
				Output:    testMsg1,
				State:     msg.StatusCompleted,
				Timestamp: utils.FormatNowUTCMilli(),
			}

			resp = req.CreateResponse(actStat, nil)

			//replyTo.SendResponse(msg.ThingID, msg.Name, output, msg.CorrelationID)
		case wot.OpQueryAllActions:
			// include an error status to ensure encode/decode of an error status works
			actStat := map[string]msg.ResponseMessage{
				actionKey: {
					ThingID:   req.ThingID,
					Name:      actionKey,
					Output:    testMsg1,
					State:     msg.StatusCompleted,
					Timestamp: utils.FormatNowUTCMilli(),
				},
				"action-2": {
					ThingID: req.ThingID,
					Name:    "action-2",
					Error: &msg.ErrorValue{
						Status: http.StatusBadRequest,
						Type:   "http://testerror/",
						Title:  "Testing error",
						Detail: "test error detail",
					},
					State:     msg.StatusFailed,
					Timestamp: utils.FormatNowUTCMilli(),
				},
				"action-3": {
					ThingID:   req.ThingID,
					Name:      "action-3",
					Output:    "other output",
					State:     msg.StatusCompleted,
					Timestamp: utils.FormatNowUTCMilli(),
				}}
			// the action status map is the payload for the action response.
			// the action response itself is also an action status object.
			resp = req.CreateResponse(actStat, nil)
		default:
			resp = req.CreateResponse(nil, errors.New("unexpected response "+req.Operation))
		}
		return replyTo(resp)
	}

	// 1. start the servers
	testEnv, cancelFn := StartTestEnv(defaultProtocol)
	defer cancelFn()
	testEnv.Server.SetRequestSink(requestHandler)

	// 2. connect as a consumer
	co1, cc1, _ := testEnv.NewConsumerClient(testClientID1, authn.ClientRoleViewer, nil)
	defer cc1.Close()

	// 3. Query action status
	var status msg.ResponseMessage
	err := co1.Rpc(wot.OpQueryAction, thingID, actionKey, nil, &status)
	require.NoError(t, err)
	require.Equal(t, thingID, status.ThingID)
	require.Equal(t, actionKey, status.Name)

	// 4. Query all actions
	var statusMap map[string]msg.ResponseMessage
	err = co1.Rpc(wot.OpQueryAllActions, thingID, actionKey, nil, &statusMap)
	require.NoError(t, err)
	require.Equal(t, 3, len(statusMap))
}
