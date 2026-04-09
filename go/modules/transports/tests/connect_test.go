package transporttests

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAgentID1 = "agent1"
const testClientID1 = "client1"

var testProtocol = transports.ProtocolTypeHiveotGrpc

var testProtocols = []string{
	transports.ProtocolTypeHiveotSsesc,
	transports.ProtocolTypeHiveotGrpc,
	transports.ProtocolTypeHiveotWebsocket,
	transports.ProtocolTypeWotWebsocket,
}

// TestMain sets logging
func TestMain(m *testing.M) {
	utils.SetLogging("info", "")
	result := m.Run()
	os.Exit(result)
}

func TestConnectAllProtocols(t *testing.T) {
	for _, testProtocol = range testProtocols {
		t.Run("TestStartStop", TestStartStop)
		t.Run(testProtocol, TestPing)
		t.Run(testProtocol, TestReconnect)
		t.Run(testProtocol, TestServerURL)
	}
}

// test create a server and connect a client
func TestStartStop(t *testing.T) {
	t.Logf("---%s %s---\n", t.Name(), testProtocol)

	testEnv, cancelFn := testenv.StartTestEnv(testProtocol)

	defer cancelFn()
	co1, cc1, _ := testEnv.NewConsumerClient(testClientID1, authnapi.ClientRoleViewer, nil)
	defer cc1.Close()
	assert.NotNil(t, co1)

	isConnected := cc1.IsConnected()
	assert.True(t, isConnected)

	// time.Sleep(time.Millisecond)
	// cc1.Close()

	t.Log("---ending---")
}

func TestPing(t *testing.T) {
	t.Logf("---%s %s---\n", t.Name(), testProtocol)

	testEnv, cancelFn := testenv.StartTestEnv(testProtocol)
	defer cancelFn()
	// NewConsumerClient creates a client
	co1, cc1, _ := testEnv.NewConsumerClient(testClientID1, authnapi.ClientRoleViewer, nil)
	defer cc1.Close()

	err := co1.Ping()
	require.NoError(t, err)

	// var output any
	// err = co1.Rpc(td.HTOpPing, "", "", nil, &output)
	// assert.Equal(t, "pong", output)
	// assert.NoError(t, err)
}

// Auto-reconnect using hub client and server
func TestReconnect(t *testing.T) {
	t.Logf("---%s %s---\n", t.Name(), testProtocol)

	const thingID = "thing1"
	const actionKey = "action1"
	const agentID = "agent1"
	var connectEvents atomic.Int32
	var reconnectedCallback atomic.Bool

	// this test handler receives an action and returns a 'pending status',
	// it is intended to prove reconnect works.
	handleRequest := func(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
		slog.Info("Received request", "op", req.Operation)
		var err error
		// prove that the return channel is connected
		if req.Operation == td.OpInvokeAction {
			go func() {
				// send an asynchronous result after a short time
				time.Sleep(time.Millisecond * 10)
				output := req.Input

				resp := req.CreateResponse(output, nil)
				// err = c.SendResponse(resp)
				err = replyTo(resp)
				assert.NoError(t, err)
			}()
			// nothing to return yet
			return nil
		}
		err = errors.New("unexpected request")
		resp := req.CreateResponse("", err)
		// err = c.SendResponse(resp)
		err = replyTo(resp)
		return err
	}
	// start the servers and handle a request
	testEnv, cancelFn := testenv.StartTestEnv(testProtocol)
	testEnv.Server.SetRequestSink(handleRequest)
	testEnv.Server.SetNotificationSink(func(notif *msg.NotificationMessage) {
		// expect a connect-disconnect event
		connectEvents.Add(1)
		slog.Info("Notification by Server",
			slog.String("type", string(notif.AffordanceType)),
			slog.String("thingID", notif.ThingID),
			slog.String("name", notif.Name),
		)
	})
	defer cancelFn()

	// connect as consumer and give client a second to reconnect
	ctx1, cancelFn1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFn1()
	connectHandler := func(connected bool, c transports.IConnection, err error) {
		if connected {
			slog.Info("reconnected")
			cancelFn1()
			reconnectedCallback.Store(true)
		} else {
			slog.Info("disconnect")
		}
	}
	co1, cc1, _ := testEnv.NewConsumerClient(testClientID1, authnapi.ClientRoleViewer, connectHandler)
	defer cc1.Close()

	//  wait until the connection is established

	// 3. close connection server side but keep the session.
	// This should trigger auto-reconnect on the client.
	t.Log("--- force disconnecting all clients. This can log a warning ---")
	testEnv.Server.CloseAll()

	<-ctx1.Done()

	// 4. invoke an action which should return a value
	// An RPC call is the ultimate test
	var rpcArgs = "rpc test"
	var rpcResp string
	time.Sleep(time.Millisecond * 3000)
	err := co1.Rpc(td.OpInvokeAction, thingID, actionKey, &rpcArgs, &rpcResp)
	require.NoError(t, err)
	assert.Equal(t, rpcArgs, rpcResp)
	assert.GreaterOrEqual(t, 4, int(connectEvents.Load()))
	// expect the re-connected callback to be invoked
	assert.True(t, reconnectedCallback.Load())
}

// Test getting form for unknown operation
//func TestBadForm(t *testing.T) {
//	t.Logf("---%s---\n", t.Name())
//
//	_, cancelFn := StartTransportServer(nil, nil, nil)
//	defer cancelFn()
//
//	form := NewForm("bad-operation", "", "")
//	assert.Nil(t, form)
//}

// Test getting server URL
func TestServerURL(t *testing.T) {
	t.Logf("---%s %s---\n", t.Name(), testProtocol)

	testEnv, cancelFn := testenv.StartTestEnv(testProtocol)
	defer cancelFn()
	serverURL := testEnv.Server.GetConnectURL()
	protocolType, subProtocol := testEnv.Server.GetProtocolType()
	_ = subProtocol
	_, err := url.Parse(serverURL)
	require.NoError(t, err)
	require.Equal(t, testProtocol, protocolType)
}
