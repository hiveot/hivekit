package transporttests

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/clients"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAgentID1 = "agent1"
const testClientID1 = "client1"

var testProtocol = transport.ProtocolTypeHiveotGrpc

var testProtocols = []string{
	transport.ProtocolTypeHiveotSsesc,
	transport.ProtocolTypeHiveotGrpc,
	transport.ProtocolTypeHiveotWebsocket,
	transport.ProtocolTypeWotWebsocket,
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
		t.Run(testProtocol, TestPingClientCert)
		t.Run(testProtocol, TestReconnect)
		t.Run(testProtocol, TestServerURL)
	}
}

// test create a server and connect a client
func TestStartStop(t *testing.T) {
	t.Logf("---%s %s---\n", t.Name(), testProtocol)

	// testenv might still start the httpserver - fixme: use on-demand factory
	testEnv, cancelFn := testenv.StartTestEnv(testProtocol)

	defer cancelFn()
	co1, cc1, _ := testEnv.NewConnectedConsumer(testClientID1, authn.ClientRoleViewer, false)
	defer cc1.Close()
	assert.NotNil(t, co1)

	status := cc1.GetConnectionStatus()
	assert.Equal(t, transport.StatusConnected, status)

	// time.Sleep(time.Millisecond)
	// cc1.Close()

	t.Log("---ending---")
}

// Run a ping test to verify a client-server connection using the test protocol
func TestPing(t *testing.T) {
	t.Logf("---%s %s---\n", t.Name(), testProtocol)

	testEnv, cancelFn := testenv.StartTestEnv(testProtocol)
	defer cancelFn()
	// NewConsumerClient creates a client
	co1, cc1, _ := testEnv.NewConnectedConsumer(testClientID1, authn.ClientRoleViewer, false)
	defer cc1.Close()

	err := co1.Ping()
	require.NoError(t, err)
}

// Run a ping test with client cert auth for the given test protocol
func TestPingClientCert(t *testing.T) {
	t.Logf("---%s %s---\n", t.Name(), testProtocol)

	testEnv, cancelFn := testenv.StartTestEnv(testProtocol)
	defer cancelFn()

	// ensure the test client account exists
	err := testEnv.TestAuthn.AddClient(testClientID1, "test", authn.ClientRoleViewer)

	// NewConsumerClient creates a client
	// create a connection to the test server
	cl, err := clients.NewTransportClient(
		testEnv.ServerProtocol, testEnv.ServerURL, testEnv.CertBundle.CaCert)
	require.NoError(t, err)
	cl.SetTimeout(time.Minute)
	err = cl.AuthenticateWithClientCert(testEnv.CertBundle.ClientCert)
	require.NoError(t, err)
	err = cl.Connect()
	require.NoError(t, err)

	cl.SetTimeout(time.Minute)
	defer cl.Close()

	// all hiveot transport handle a ping message
	req := msg.NewRequestMessage(td.HTOpPing, "", "", nil)
	err = cl.HandleRequest(req, func(resp *msg.ResponseMessage) error {
		slog.Info("Received response")
		return nil
	})
	require.NoError(t, err)

	status := cl.GetConnectionStatus()
	assert.Equal(t, transport.StatusConnected, status)
}

// Auto-reconnect using hub client and server
func TestReconnect(t *testing.T) {
	t.Logf("---%s %s---\n", t.Name(), testProtocol)

	const thingID = "thing1"
	const actionKey = "action1"
	const agentID = "agent1"
	var connectEvents atomic.Int32

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
				resp.SenderID = agentID
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
	co1, cc1, _ := testEnv.NewConnectedConsumer(testClientID1, authn.ClientRoleViewer, true)
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
	err := co1.InvokeAction(thingID, actionKey, &rpcArgs, &rpcResp)
	require.NoError(t, err)
	assert.Equal(t, rpcArgs, rpcResp)
	assert.GreaterOrEqual(t, 4, int(connectEvents.Load()))
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
	assert.NotEmpty(t, serverURL)
}
