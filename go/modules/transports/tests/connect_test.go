package tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/gocore/servers/httpbasic"
	"github.com/hiveot/hivekit/go/lib/clients/authclient"
	"github.com/hiveot/hivekit/go/lib/logging"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/services/certs/service/selfsigned"
	"github.com/hiveot/hivekit/go/modules/transports"
	hiveotsse "github.com/hiveot/hivekit/go/modules/transports/hiveotsse"
	sseapi "github.com/hiveot/hivekit/go/modules/transports/hiveotsse/api"
	hiveotssemodule "github.com/hiveot/hivekit/go/modules/transports/hiveotsse/module"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic/httpbasicapi"
	httpbasicmodule "github.com/hiveot/hivekit/go/modules/transports/httpbasic/module"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver/module"
	wssserver "github.com/hiveot/hivekit/go/modules/transports/wss"
	wssapi "github.com/hiveot/hivekit/go/modules/transports/wss/api"
	wssmodule "github.com/hiveot/hivekit/go/modules/transports/wss/module"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils/authn"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTimeout = time.Second * 300
const testAgentID1 = "agent1"
const testClientID1 = "client1"
const testServerHttpPort = 9445
const testServerHttpURL = "https://localhost:9445"

// const testServerHiveotHttpBasicURL = "wss://localhost:9445" + server.DefaultHiveotHttpBasicPath
const testServerHiveotSseURL = "sse://localhost:9445" + hiveotsse.DefaultHiveotSsePath
const testServerHiveotWssURL = "wss://localhost:9445" + wssserver.DefaultHiveotWssPath

//const testServerMqttWssURL = "mqtts://localhost:9447"

var defaultProtocol = transports.ProtocolTypeHiveotSSE

// var defaultProtocol = transports.ProtocolTypeWSS

var transportServer transports.ITransportModule
var dummyAuthenticator *authn.DummyAuthenticator
var certBundle = selfsigned.CreateTestCertBundle()

// NewClient creates a new connected client with the given client ID. The
// transport server must be started first.
//
// This uses the server to generate an auth token.
// This panics if a client cannot be created.
// ClientID is only used for logging
func NewTestClient(clientID string) (transports.IClientConnection, string) {
	caCert := certBundle.CaCert
	fullURL := testServerHttpURL
	token := dummyAuthenticator.AddClient(clientID, clientID)
	var cl transports.IClientConnection
	var sink modules.IHiveModule

	switch defaultProtocol {
	case transports.ProtocolTypeHiveotSSE:
		fullURL = testServerHiveotSseURL
		cl = sseapi.NewHiveotSseClient(fullURL, clientID, caCert, sink, testTimeout)

	case transports.ProtocolTypeWotWSS:
		fullURL = testServerHiveotWssURL
		cl = wssapi.NewWotWssClient(fullURL, clientID, caCert, sink, testTimeout)

	case transports.ProtocolTypeHTTPBasic:
		caCert := certBundle.CaCert
		fullURL = testServerHttpURL
		cl = httpbasicapi.NewHttpBasicClient(
			fullURL, clientID, caCert, sink, nil, testTimeout)

		//case transports.ProtocolTypeWotMQTTWSS:
		//	fullURL = testServerMqttWssURL
	}
	err := cl.ConnectWithToken(token)
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
func NewAgent(clientID string) (transports.IClientConnection, *transports.Agent, string) {
	cc, token := NewTestClient(clientID)

	agent := transports.NewAgent(cc, nil, nil, nil, nil, testTimeout)
	return cc, agent, token
}

// NewConsumer creates a new connected consumer client with the given ID.
// The transport server must be started first.
//
// This uses the clientID as password
// This panics if a client cannot be created
func NewConsumer(clientID string) (
	transports.IClientConnection, *transports.Consumer, string) {

	cc, token := NewTestClient(clientID)
	co := transports.NewConsumer(cc, testTimeout)
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

// start the 'defaultProtocol' transport server. This is one of http-basic,
// http-sse, wot or hiveot websocket.
// This panics if the server cannot be created
func StartTransportModule(
	notifHandler msg.NotificationHandler,
	reqHandler msg.RequestHandler,
	respHandler msg.ResponseHandler,
) (srv transports.ITransportModule, cancelFunc func()) {

	caCert := certBundle.CaCert
	serverCert := certBundle.ServerCert
	dummyAuthenticator = authn.NewDummyAuthenticator()
	if reqHandler == nil {
		reqHandler = DummyRequestHandler
	}
	if respHandler == nil {
		respHandler = DummyResponseHandler
	}
	// cert uses localhost
	cfg := httpserver.NewHttpServerConfig()
	// cfg.Address = fmt.Sprintf("%s:%d", certBundle.ServerAddr, testServerHttpPort)
	cfg.Address = certBundle.ServerAddr
	cfg.Port = testServerHttpPort
	cfg.CaCert = caCert
	cfg.ServerCert = serverCert

	httpServer := module.NewHttpServerModule("", cfg)
	err := httpServer.Start()

	// tlsServer, httpRouter := tlsserver.NewTLSServer(
	// 	certBundle.ServerAddr, testServerHttpPort, serverCert, caCert)
	// err := tlsServer.Start()
	if err != nil {
		panic("unable to start TLS server: " + err.Error())
	}

	switch defaultProtocol {
	case transports.ProtocolTypeHTTPBasic:

		// httpbasic is required for login
		transportServer = httpbasicmodule.NewHttpBasicModule(
			httpServer, nil, dummyAuthenticator)
		err = transportServer.Start()
		// http only, no subprotocol bindings

	case transports.ProtocolTypeHiveotSSE:
		transportServer := hiveotssemodule.NewHiveotSseModule(
			httpServer, nil, nil)
		err = transportServer.Start()

	case transports.ProtocolTypeWotWSS:
		transportServer := wssmodule.NewWotWssModule(httpServer, nil)
		err = transportServer.Start()

	default:
		err = errors.New("unknown protocol name: " + defaultProtocol)
	}

	if err != nil {
		panic("Unable to create protocol server: " + err.Error())
	}
	//transportServer.SetRequestHandler(cm.AddConnection)
	//transportServer.SetMessageHandler(cm.AddConnection)

	return transportServer, func() {
		if transportServer != nil {
			transportServer.Stop()
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

	srv, cancelFn := StartTransportModule(nil, nil, nil)
	_ = srv
	defer cancelFn()
	cc1, co1, _ := NewConsumer(testClientID1)
	defer cc1.Disconnect()
	assert.NotNil(t, co1)

	isConnected := cc1.IsConnected()
	assert.True(t, isConnected)
}

// login/refresh use the http-basic or http-sse binding
func TestLoginRefresh(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	srv, cancelFn := StartTransportModule(nil, nil, nil)
	defer cancelFn()
	// ensure the client exists
	cc1, co1, token1 := NewConsumer(testClientID1)
	_ = co1

	isConnected := cc1.IsConnected()
	// FIXME: separate auth from the client
	//assert.False(t, isConnected)
	//
	//token1, err := cc1.ConnectWithPassword(testClientID1)
	//require.NoError(t, err)
	//require.NotEmpty(t, token1)

	// check if both client and server have the connection ID
	// the server prefixes it with clientID- to ensure no client can steal another's ID
	// removed this test as it is only important for http/sse. Other tests will fail
	// if they are incorrect.
	//cid1 := co1.GetConnectionID()
	//assert.NotEmpty(t, cid1)
	//srvConn := cm.GetConnectionByClientID(testClientID1)
	//require.NotNil(t, srvConn)
	//cid1server := srvConn.GetConnectionID()
	//assert.Equal(t, testClientID1+"-"+cid1, cid1server)

	isConnected = cc1.IsConnected()
	assert.True(t, isConnected)

	parts, err := url.Parse(srv.GetConnectURL())
	require.NoError(t, err)
	authCl := authclient.NewAuthClient(parts.Host, certBundle.CaCert, testTimeout)
	token2, err := authCl.RefreshToken(token1)

	// refresh should succeed
	//token2, err := co1.RefreshToken(token1)
	require.NoError(t, err)
	require.NotEmpty(t, token2)

	// end the connection
	cc1.Disconnect()
	time.Sleep(time.Millisecond * 1)

	// should be able to reconnect with the new token
	// NOTE: the runtime session manager doesn't allow this as
	// the session no longer exists, but the dummyAuthenticator doesn't care.
	err = cc1.ConnectWithToken(token2)
	require.NoError(t, err)

	//token3, err := co1.RefreshToken(token2)
	token3, err := authCl.RefreshToken(token2)
	assert.NoError(t, err)
	assert.NotEmpty(t, token3)

	// end the session
	cc1.Disconnect()
}

func TestLogout(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	srv, cancelFn := StartTransportModule(nil, nil, nil)
	_ = srv
	defer cancelFn()

	// check if this test still works with a valid login
	cc1, co1, token1 := NewConsumer(testClientID1)
	_ = cc1
	_ = co1
	defer co1.Disconnect()
	assert.NotEmpty(t, token1)

	// logout
	authCl := authclient.NewAuthClientFromConnection(cc1, token1)
	err := authCl.Logout()

	//authenticator.Logout(cc1, "")
	//err := co1.Logout()
	t.Log("logged out, some warnings are expected next")
	assert.NoError(t, err)

	// This causes Refresh to fail
	token2, err := authCl.RefreshToken(token1)
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
	srv, cancelFn := StartTransportModule(nil, nil, nil)
	defer cancelFn()
	cc1, co1, token1 := NewConsumer(testClientID1)
	_ = co1
	_ = token1
	defer cc1.Disconnect()

	// set the token
	t.Log("Expecting SetBearerToken('bad-token') to fail")
	err := cc1.ConnectWithToken("bad-token")
	require.Error(t, err)

	// reconnect with a valid token and connect with a bad client-id
	err = cc1.ConnectWithToken(token1)
	assert.NoError(t, err)
	parts, _ := url.Parse(srv.GetConnectURL())
	authCl := authclient.NewAuthClient(
		parts.Host, certBundle.CaCert, testTimeout)
	validToken, err := authCl.RefreshToken(token1)
	//validToken, err := co1.RefreshToken(token1)
	assert.NoError(t, err)
	assert.NotEmpty(t, validToken)
	cc1.Disconnect()
}

// Auto-reconnect using hub client and server
func TestReconnect(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	const thingID = "thing1"
	const actionKey = "action1"
	const agentID = "agent1"
	var reconnectedCallback atomic.Bool
	var dThingID = td.MakeDigiTwinThingID(agentID, thingID)
	var srv transports.ITransportModule
	var cancelFn func()

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
	// start the servers and connect as a client
	srv, cancelFn = StartTransportModule(nil, handleRequest, nil)
	defer cancelFn()

	// connect as client
	cc1, co1, _ := NewConsumer(testClientID1)
	//token := dummyAuthenticator.CreateSessionToken(testClientID1, "", 0)
	//err := cc1.ConnectWithToken(token)
	//require.NoError(t, err)
	defer cc1.Disconnect()

	//  wait until the connection is established

	// 3. close connection server side but keep the session.
	// This should trigger auto-reconnect on the client.
	t.Log("--- force disconnecting all clients ---")
	srv.CloseAll()

	// give client time to reconnect
	ctx1, cancelFn1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFn1()
	cc1.SetConnectHandler(func(connected bool, err error, c transports.IConnection) {
		if connected {
			cancelFn1()
			reconnectedCallback.Store(true)
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

func TestPing(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	srv, cancelFn := StartTransportModule(nil, nil, nil)
	_ = srv
	defer cancelFn()
	cc1, co1, _ := NewConsumer(testClientID1)
	defer cc1.Disconnect()

	//_, err := cc1.ConnectWithPassword(testClientID1)
	//require.NoError(t, err)

	err := co1.Ping()
	assert.NoError(t, err)

	// FIXME: SSE server sends ping event but it isn't received until later???

	var output any
	err = co1.Rpc(wot.HTOpPing, "", "", nil, &output)
	assert.Equal(t, "pong", output)
	assert.NoError(t, err)
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

	srv, cancelFn := StartTransportModule(nil, nil, nil)
	defer cancelFn()
	serverURL := srv.GetConnectURL()
	_, err := url.Parse(serverURL)
	require.NoError(t, err)
}

// Test ping/login/refresh using http-basic
func TestHttpBasic(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const testPass = "password-test"
	var clientSink modules.IHiveModule

	// all transport servers use http-basic
	// this also creates a dummy authenticator
	srv, cancelFn := StartTransportModule(nil, nil, nil)
	_ = srv
	defer cancelFn()
	token1 := dummyAuthenticator.AddClient(testClientID1, testPass)
	_ = token1

	// connect using http-basic
	serverURL := srv.GetConnectURL()

	cl := httpbasicapi.NewHttpBasicClient(serverURL, testClientID1,
		certBundle.CaCert, clientSink, nil, time.Second)
	//err := htb.ConnectWithToken(token)
	//require.NoError(t, err)

	// 1: Ping
	_, _, code, err := cl.Send(http.MethodGet, httpbasic.HttpGetPingPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, code)

	// 2: Login
	loginBody := fmt.Sprintf(`{"login":"%s", "password":"%s"}`, testClientID1, testPass)
	body, headers, code, err := cl.Send(http.MethodPost, httpbasic.HttpPostLoginPath, []byte(loginBody))
	require.NoError(t, err)
	require.Equal(t, 200, code)
	require.NotEmpty(t, body)
	require.NotEmpty(t, headers)
	var token2 string
	err = json.Unmarshal(body, &token2)
	require.NoError(t, err)

	// 3: Refresh using auth token
	err = cl.SetBearerToken(token2)
	assert.NoError(t, err)
	refreshBody, _ := json.Marshal(token2)
	body, headers, code, err = cl.Send(http.MethodPost, httpbasic.HttpPostRefreshPath, refreshBody)
	require.NoError(t, err)
	require.Equal(t, 200, code)
	require.NotEmpty(t, body)
	require.NotEmpty(t, headers)

}
