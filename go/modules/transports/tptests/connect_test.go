package tptests

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/lib/logging"
	"github.com/hiveot/hivekit/go/modules/certs/module/selfsigned"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAgentID1 = "agent1"
const testClientID1 = "client1"

// server endpoint/protocol used
var defaultProtocol = transports.ProtocolTypeHiveotSSE

// var defaultProtocol = transports.ProtocolTypeWotWSS

var certBundle = selfsigned.CreateTestCertBundle(utils.KeyTypeED25519)

// Create a new form for the given operation
// This uses the default protocol to generate the Form
//func NewForm(op, thingID, name string) *td.Form {
//	switch defaultProtocol {
//
//	}
//	form := transportServer.GetForm(op, thingID, name)
//	return form
//}

func DummyRequestHandler(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	var output any
	var err error
	slog.Info("DummyRequestHandler: Received request", "op", req.Operation)
	//if req.Operation == wot.HTOpRefresh {
	//	oldToken := req.ToString(0)
	//	output, err = dummyAuthenticator.RefreshToken(req.SenderID, oldToken)
	//} else if req.Operation == wot.HTOpLogout {
	//	dummyAuthenticator.Logout(c.GetClientID())
	//} else {
	output = req.Input // echo
	//}
	resp := req.CreateResponse(output, err)
	err = replyTo(resp)
	return err
}

func DummyResponseHandler(response *msg.ResponseMessage) error {

	slog.Info("DummyResponse: Received response", "op", response.Operation)
	return nil
}

// TestMain sets logging
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	result := m.Run()
	os.Exit(result)
}

// test create a server and connect a client
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	testEnv, cancelFn := StartTestEnv(defaultProtocol)

	defer cancelFn()
	co1, cc1, _ := testEnv.NewConsumerClient(testClientID1, transports.ClientRoleViewer, nil)
	defer cc1.Close()
	assert.NotNil(t, co1)

	isConnected := cc1.IsConnected()
	assert.True(t, isConnected)
}

func TestPing(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	testEnv, cancelFn := StartTestEnv(defaultProtocol)
	defer cancelFn()
	co1, cc1, _ := testEnv.NewConsumerClient(testClientID1, transports.ClientRoleViewer, nil)
	defer cc1.Close()

	for i := 0; i < 1; i++ {
		err := co1.Ping()
		require.NoError(t, err)
	}

	// FIXME: SSE server sends ping event but it isn't received until later???

	// var output any
	// err = co1.Rpc(wot.HTOpPing, "", "", nil, &output)
	// assert.Equal(t, "pong", output)
	// assert.NoError(t, err)
}

// login/refresh
// func TestLoginRefresh(t *testing.T) {
// 	t.Logf("---%s---\n", t.Name())
// 	testPass := "pass1"
// 	srv, tpauthn, cancelFn := StartTransportModule(nil)
// 	require.NotNil(t, srv)
// 	defer cancelFn()

// 	serverURL := srv.GetConnectURL()
// 	authnClient := authnclient.NewAuthnHttpClient(serverURL, certBundle.CaCert)

// 	// 1: Login
// 	tpauthn.AddClient(testClientID1, testPass)
// 	token, err := authnClient.LoginWithPassword(testClientID1, testPass)
// 	require.NoError(t, err)
// 	require.NotEmpty(t, token)

// 	// 2: Refresh using auth token
// 	token2, err := authnClient.RefreshToken(token)
// 	require.NoError(t, err)
// 	require.NotEmpty(t, token2)

// 	// end the connection
// 	authnClient.Close()
// 	time.Sleep(time.Millisecond * 1)

// 	// should be able to reconnect with the new token and refresh.
// 	parts, _ := url.Parse(serverURL)
// 	cl2 := tlsclient.NewTLSClient(parts.Host, nil, certBundle.CaCert, 0)
// 	err = cl2.ConnectWithToken(testClientID1, token2)
// 	require.NoError(t, err)

// 	//token3, err := co1.RefreshToken(token2)
// 	token3, err := authnclient.RefreshToken(cl2, testClientID1, token2)
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, token3)

// 	// end the session
// 	cl2.Close()
// }

// func TestLogout(t *testing.T) {
// 	t.Logf("---%s---\n", t.Name())

// 	srv, tpauthn, cancelFn := StartTransportModule(nil)
// 	_ = srv
// 	defer cancelFn()

// 	// check if this test still works with a valid login
//	cc1, co1, _ := NewTestConsumer(testClientID1, srv.GetConnectURL(), tpauthn)
// 	_ = cc1
// 	_ = co1
// 	defer co1.Stop()
// 	assert.NotEmpty(t, token1)

// 	// logout
// 	serverURL := srv.GetConnectURL()
// 	authnClient := authnclient.NewAuthnHttpClient(serverURL, certBundle.CaCert)
// 	authnClient.ConnectWithToken(testClientID1, token1)
// 	err := authnClient.Logout(token1)
// 	assert.NoError(t, err)

// 	//authenticator.Logout(cc1, "")
// 	//err := co1.Logout()
// 	t.Log(">>> Logged out, an unauthorized error is expected next.")

// 	// This causes Refresh to fail
// 	token2, err := authnClient.RefreshToken(token1)
// 	//token2, err := co1.RefreshToken(token1)
// 	assert.Error(t, err)
// 	assert.Empty(t, token2)
// }

//func TestBadLogin(t *testing.T) {
//	t.Logf("---%s---\n", t.Name())
//
//	srv, cancelFn := StartTransportServer(nil, nil)
//	defer cancelFn()
//
//	cc1, co1, _ := NewConsumer(testClientID1, srv.GetForm)
//
//	// check if this test still works with a valid login
//	token1, err := cc1.ConnectWithPassword(testClientID1)
//	assert.NoError(t, err)
//
//	// failed logins
//	t.Log("Expecting ConnectWithPassword to fail")
//	token2, err := cc1.ConnectWithPassword("bad-pass")
//	assert.Error(t, err)
//	assert.Empty(t, token2)
//
//	// can't refresh when no longer connected
//	t.Log("Expecting RefreshToken to fail")
//	token4, err := co1.RefreshToken(token1)
//	assert.Error(t, err)
//	assert.Empty(t, token4)
//
//	// disconnect should always succeed
//	cc1.Disconnect()
//
//	// bad client ID
//	t.Log("Expecting ConnectWithPassword('BadID') to fail")
//	cc2, _, _ := NewConsumer("badID", srv.GetForm)
//	token5, err := cc2.ConnectWithPassword(testClientID1)
//	assert.Error(t, err)
//	assert.Empty(t, token5)
//}

// func TestBadRefresh(t *testing.T) {
// 	t.Logf("---%s---\n", t.Name())
// 	srv, tpauthn, cancelFn := StartTransportModule(nil)
// 	defer cancelFn()
// 	cc1, co1, token1 := NewTestConsumer(tpauthn, testClientID1)
// 	_ = co1
// 	_ = token1
// 	defer cc1.Close()

// 	// set the token
// 	t.Log("Expecting SetBearerToken('bad-token') to fail")
// 	err := cc1.ConnectWithToken(testClientID1, "bad-token")
// 	require.Error(t, err)

// 	// reconnect with a valid token and connect with a bad client-id
// 	err = cc1.ConnectWithToken(testClientID1, token1)
// 	assert.NoError(t, err)

// 	serverURL := srv.GetConnectURL()
// 	authCl := authnclient.NewAuthnHttpClient(serverURL, certBundle.CaCert)
// 	authCl.ConnectWithToken(testClientID1, token1)
// 	validToken, err := authCl.RefreshToken(token1)
// 	//validToken, err := co1.RefreshToken(token1)
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, validToken)
// 	cc1.Close()
// }

// Auto-reconnect using hub client and server
func TestReconnect(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	const thingID = "thing1"
	const actionKey = "action1"
	const agentID = "agent1"
	var reconnectedCallback atomic.Bool
	var dThingID = td.MakeDigiTwinThingID(agentID, thingID)

	// this test handler receives an action and returns a 'pending status',
	// it is intended to prove reconnect works.
	handleRequest := func(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
		slog.Info("Received request", "op", req.Operation)
		var err error
		// prove that the return channel is connected
		if req.Operation == wot.OpInvokeAction {
			go func() {
				// send an asynchronous result after a short time
				time.Sleep(time.Millisecond * 10)
				// require.NotNil(t, c, "client doesnt have a connection")
				output := req.Input
				// cinfo := c.GetConnectionInfo()
				// c2 := srv.GetConnectionByConnectionID(cinfo.ClientID, cinfo.ConnectionID)
				// assert.NotEmpty(t, c2)
				resp, as := req.CreateActionResponse(
					req.CorrelationID, msg.StatusCompleted, output, nil)
				_ = as
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
	testEnv, cancelFn := StartTestEnv(defaultProtocol)
	testEnv.Server.SetRequestSink(handleRequest)
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
	co1, cc1, _ := testEnv.NewConsumerClient(testClientID1, transports.ClientRoleViewer, connectHandler)
	defer cc1.Close()

	//  wait until the connection is established

	// 3. close connection server side but keep the session.
	// This should trigger auto-reconnect on the client.
	t.Log("--- force disconnecting all clients ---")
	testEnv.Server.CloseAll()

	<-ctx1.Done()

	// 4. invoke an action which should return a value
	// An RPC call is the ultimate test
	var rpcArgs = "rpc test"
	var rpcResp string
	time.Sleep(time.Millisecond * 3000)
	err := co1.Rpc(wot.OpInvokeAction, dThingID, actionKey, &rpcArgs, &rpcResp)
	require.NoError(t, err)
	assert.Equal(t, rpcArgs, rpcResp)

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
	t.Logf("---%s---\n", t.Name())

	testEnv, cancelFn := StartTestEnv(defaultProtocol)
	defer cancelFn()
	serverURL := testEnv.Server.GetConnectURL()
	_, err := url.Parse(serverURL)
	require.NoError(t, err)
}
