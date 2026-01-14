package tests

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
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/services/certs/service/selfsigned"
	"github.com/hiveot/hivekit/go/modules/transports"
	authnapi "github.com/hiveot/hivekit/go/modules/transports/authn/api"
	authnmodule "github.com/hiveot/hivekit/go/modules/transports/authn/module"
	hiveotsse "github.com/hiveot/hivekit/go/modules/transports/hiveotsse"
	sseapi "github.com/hiveot/hivekit/go/modules/transports/hiveotsse/api"
	hiveotssemodule "github.com/hiveot/hivekit/go/modules/transports/hiveotsse/module"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic/httpbasicapi"
	httpbasicmodule "github.com/hiveot/hivekit/go/modules/transports/httpbasic/module"
	"github.com/hiveot/hivekit/go/modules/transports/httptransport"
	"github.com/hiveot/hivekit/go/modules/transports/httptransport/httpapi"
	"github.com/hiveot/hivekit/go/modules/transports/httptransport/module"
	wssserver "github.com/hiveot/hivekit/go/modules/transports/wss"
	wssapi "github.com/hiveot/hivekit/go/modules/transports/wss/api"
	wssmodule "github.com/hiveot/hivekit/go/modules/transports/wss/module"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const appID = "connect-test"
const testTimeout = time.Second * 300
const testAgentID1 = "agent1"
const testClientID1 = "client1"
const testServerHttpPort = 9445
const testServerHttpURL = "https://localhost:9445"

// const testServerHiveotHttpBasicURL = "wss://localhost:9445" + server.DefaultHiveotHttpBasicPath
const testServerHiveotSseURL = "sse://localhost:9445" + hiveotsse.DefaultHiveotSsePath
const testServerHiveotWssURL = "wss://localhost:9445" + wssserver.DefaultHiveotWssPath

//const testServerMqttWssURL = "mqtts://localhost:9447"

// var defaultProtocol = transports.ProtocolTypeHiveotSSE
var defaultProtocol = transports.ProtocolTypeWotWSS

// var dummyAuthenticator *authnapi.DummyAuthenticator
var certBundle = selfsigned.CreateTestCertBundle()

// NewClient creates a new connected client with the given client ID. The
// transport server must be started first.
//
// This uses the server to generate an auth token.
// This panics if a client cannot be created.
// ClientID is only used for logging
func NewTestClient(tpauthn *authnapi.DummyAuthenticator, clientID string) (transports.IClientConnection, string) {
	caCert := certBundle.CaCert
	fullURL := testServerHttpURL
	token := tpauthn.AddClient(clientID, clientID)
	var cl transports.IClientConnection
	var sink modules.IHiveModule

	switch defaultProtocol {
	case transports.ProtocolTypeHiveotSSE:
		fullURL = testServerHiveotSseURL
		cl = sseapi.NewHiveotSseClient(fullURL, caCert, sink, testTimeout)

	case transports.ProtocolTypeWotWSS:
		fullURL = testServerHiveotWssURL
		cl = wssapi.NewWotWssClient(fullURL, caCert, sink, testTimeout)

	case transports.ProtocolTypeHTTPBasic:
		caCert := certBundle.CaCert
		fullURL = testServerHttpURL
		cl = httpbasicapi.NewHttpBasicClient(
			fullURL, caCert, sink, nil, testTimeout)

		//case transports.ProtocolTypeWotMQTTWSS:
		//	fullURL = testServerMqttWssURL
	}
	err := cl.ConnectWithToken(clientID, token)
	if err != nil {
		panic("NewClient failed:" + err.Error())
	}
	return cl, token
}

// NewAgent creates a new connected agent client with the given ID. The
// transport server must be started first.
//
// This uses the clientID as password
// This panics if a client cannot be created
func NewAgent(tpauthn *authnapi.DummyAuthenticator, clientID string) (transports.IClientConnection, *transports.Agent, string) {
	cc, token := NewTestClient(tpauthn, clientID)

	agent := transports.NewAgent(clientID, cc, nil, nil, nil, nil, testTimeout)
	return cc, agent, token
}

// NewConsumer creates a new connected consumer client with the given ID.
// The transport server must be started first.
//
// This uses the clientID as password
// This panics if a client cannot be created
func NewConsumer(tpauthn *authnapi.DummyAuthenticator, clientID string) (
	transports.IClientConnection, *transports.Consumer, string) {

	cc, token := NewTestClient(tpauthn, clientID)
	co := transports.NewConsumer(appID, cc, testTimeout)
	return cc, co, token
}

// Create a new form for the given operation
// This uses the default protocol to generate the Form
//func NewForm(op, thingID, name string) *td.Form {
//	switch defaultProtocol {
//
//	}
//	form := transportServer.GetForm(op, thingID, name)
//	return form
//}

// start the 'defaultProtocol' transport server with dummy authenticator.
// This is one of http-basic,
// http-sse, wot or hiveot websocket.
// This panics if the server cannot be created
func StartTransportModule(sink modules.IHiveModule) (
	srv transports.ITransportModule, authenticator *authnapi.DummyAuthenticator, cancelFunc func()) {

	caCert := certBundle.CaCert
	serverCert := certBundle.ServerCert
	dummyAuthenticator := authnapi.NewDummyAuthenticator()

	// cert uses localhost
	cfg := httptransport.NewHttpServerConfig(
		certBundle.ServerAddr, testServerHttpPort, serverCert, caCert,
		dummyAuthenticator.ValidateToken)
	// cfg.Address = fmt.Sprintf("%s:%d", certBundle.ServerAddr, testServerHttpPort)

	httpServer := module.NewHttpServerModule("", cfg)
	err := httpServer.Start()

	// tlsServer, httpRouter := tlsserver.NewTLSServer(
	// 	certBundle.ServerAddr, testServerHttpPort, serverCert, caCert)
	// err := tlsServer.Start()
	if err != nil {
		panic("unable to start TLS server: " + err.Error())
	}

	// start the auth server to test login/refresh/logout
	authnModule := authnmodule.NewAuthnModule(httpServer, dummyAuthenticator)
	err = authnModule.Start()
	if err != nil {
		panic("unable to start Authn server: " + err.Error())
	}

	switch defaultProtocol {
	case transports.ProtocolTypeHTTPBasic:

		srv = httpbasicmodule.NewHttpBasicModule(httpServer, sink)
		err = srv.Start()
		// http only, no subprotocol bindings

	case transports.ProtocolTypeHiveotSSE:
		srv = hiveotssemodule.NewHiveotSseModule(httpServer, sink, nil)
		err = srv.Start()

	case transports.ProtocolTypeWotWSS:
		srv = wssmodule.NewWotWssModule(httpServer, sink)
		err = srv.Start()

	default:
		err = errors.New("unknown protocol name: " + defaultProtocol)
	}

	if err != nil {
		panic("Unable to create protocol server: " + err.Error())
	}
	//transportServer.SetRequestHandler(cm.AddConnection)
	//transportServer.SetMessageHandler(cm.AddConnection)

	return srv, dummyAuthenticator, func() {
		if srv != nil {
			srv.Stop()
		}
		if authnModule != nil {
			authnModule.Stop()
		}
		if httpServer != nil {
			httpServer.Stop()
		}
	}
}

// func DummyNotificationHandler(notification transports.NotificationMessage) {
//
//		slog.Info("DummyNotificationHandler: Received notification", "op", notification.Operation)
//		//replyTo.SendResponse(msg.ThingID, msg.Name, "result", msg.CorrelationID)
//	}
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

	srv, tpauthn, cancelFn := StartTransportModule(nil)
	_ = srv
	defer cancelFn()
	cc1, co1, _ := NewConsumer(tpauthn, testClientID1)
	defer cc1.Close()
	assert.NotNil(t, co1)

	isConnected := cc1.IsConnected()
	assert.True(t, isConnected)
}

func TestPing(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	srv, tpauthn, cancelFn := StartTransportModule(nil)
	_ = srv
	defer cancelFn()
	cc1, co1, _ := NewConsumer(tpauthn, testClientID1)
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
func TestLoginRefresh(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	testPass := "pass1"
	srv, tpauthn, cancelFn := StartTransportModule(nil)
	require.NotNil(t, srv)
	defer cancelFn()

	serverURL := srv.GetConnectURL()
	authnClient := authnapi.NewAuthnClient(serverURL, certBundle.CaCert)

	// 1: Login
	tpauthn.AddClient(testClientID1, testPass)
	token, err := authnClient.LoginWithPassword(testClientID1, testPass)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// 2: Refresh using auth token
	token2, err := authnClient.RefreshToken(token)
	require.NoError(t, err)
	require.NotEmpty(t, token2)

	// end the connection
	authnClient.Close()
	time.Sleep(time.Millisecond * 1)

	// should be able to reconnect with the new token and refresh.
	parts, _ := url.Parse(serverURL)
	cl2 := httpapi.NewTLSClient(parts.Host, nil, certBundle.CaCert, 0)
	err = cl2.ConnectWithToken(testClientID1, token2)
	require.NoError(t, err)

	//token3, err := co1.RefreshToken(token2)
	token3, err := authnapi.RefreshToken(cl2, testClientID1, token2)
	assert.NoError(t, err)
	assert.NotEmpty(t, token3)

	// end the session
	cl2.Close()
}

func TestLogout(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	srv, tpauthn, cancelFn := StartTransportModule(nil)
	_ = srv
	defer cancelFn()

	// check if this test still works with a valid login
	cc1, co1, token1 := NewConsumer(tpauthn, testClientID1)
	_ = cc1
	_ = co1
	defer co1.Stop()
	assert.NotEmpty(t, token1)

	// logout
	serverURL := srv.GetConnectURL()
	authnClient := authnapi.NewAuthnClient(serverURL, certBundle.CaCert)
	authnClient.ConnectWithToken(testClientID1, token1)
	err := authnClient.Logout(token1)
	assert.NoError(t, err)

	//authenticator.Logout(cc1, "")
	//err := co1.Logout()
	t.Log(">>> Logged out, an unauthorized error is expected next.")

	// This causes Refresh to fail
	token2, err := authnClient.RefreshToken(token1)
	//token2, err := co1.RefreshToken(token1)
	assert.Error(t, err)
	assert.Empty(t, token2)
}

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

func TestBadRefresh(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	srv, tpauthn, cancelFn := StartTransportModule(nil)
	defer cancelFn()
	cc1, co1, token1 := NewConsumer(tpauthn, testClientID1)
	_ = co1
	_ = token1
	defer cc1.Close()

	// set the token
	t.Log("Expecting SetBearerToken('bad-token') to fail")
	err := cc1.ConnectWithToken(testClientID1, "bad-token")
	require.Error(t, err)

	// reconnect with a valid token and connect with a bad client-id
	err = cc1.ConnectWithToken(testClientID1, token1)
	assert.NoError(t, err)

	serverURL := srv.GetConnectURL()
	authCl := authnapi.NewAuthnClient(serverURL, certBundle.CaCert)
	authCl.ConnectWithToken(testClientID1, token1)
	validToken, err := authCl.RefreshToken(token1)
	//validToken, err := co1.RefreshToken(token1)
	assert.NoError(t, err)
	assert.NotEmpty(t, validToken)
	cc1.Close()
}

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
	tmpSink := &modules.HiveModuleBase{}
	tmpSink.SetRequestHandler(handleRequest)
	tpm, tpauthn, cancelFn := StartTransportModule(tmpSink)
	defer cancelFn()

	// connect as consumer
	cc1, co1, _ := NewConsumer(tpauthn, testClientID1)
	defer cc1.Close()

	//  wait until the connection is established

	// 3. close connection server side but keep the session.
	// This should trigger auto-reconnect on the client.
	t.Log("--- force disconnecting all clients ---")
	tpm.CloseAll()

	// give client a second to reconnect
	ctx1, cancelFn1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFn1()
	cc1.SetConnectHandler(func(connected bool, err error, c transports.IConnection) {
		if connected {
			slog.Info("reconnected")
			cancelFn1()
			reconnectedCallback.Store(true)
		} else {
			slog.Info("disconnect")
		}
	})
	<-ctx1.Done()

	// 4. invoke an action which should return a value
	// An RPC call is the ultimate test
	var rpcArgs = "rpc test"
	var rpcResp string
	time.Sleep(time.Millisecond * 1000)
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

	srv, _, cancelFn := StartTransportModule(nil)
	defer cancelFn()
	serverURL := srv.GetConnectURL()
	_, err := url.Parse(serverURL)
	require.NoError(t, err)
}
