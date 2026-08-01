package transporttests

import (
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/transport/clients"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDeviceID1 = "device1"
const testClientID1 = "client1"

var testProtocol = api.WotWebsocketProtocolType

var testProtocols = []string{
	api.HiveotSseScProtocolType,
	api.HiveotGrpcTcpProtocolType,
	api.HiveotGrpcUnixProtocolType,
	api.HiveotWebsocketProtocolType,
	api.WotWebsocketProtocolType,
	api.HttpBasicProtocolType,
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
		t.Run("TestPing", TestPing)
		t.Run("TestPingClientCert", TestPingClientCert)
		t.Run("TestServerURL", TestServerURL)
	}
}

// test create a server and connect a client
func TestStartStop(t *testing.T) {
	slog.Warn(fmt.Sprintf("---Test: %s %s---\n", t.Name(), testProtocol))

	// testenv might still start the httpserver - fixme: use on-demand factory
	testEnv, cancelFn := testenv.StartTestEnv(testProtocol, true)

	defer cancelFn()
	co1, cc1, _ := testEnv.NewConnectedConsumer(testClientID1, authn.ClientRoleViewer)
	defer cc1.Close()
	assert.NotNil(t, co1)

	status := cc1.GetConnectionStatus()
	assert.Equal(t, api.StatusConnected, status)

	// time.Sleep(time.Millisecond)
	// cc1.Close()

	t.Log("---ending---")
}

// Run a ping test to verify a client-server connection using the test protocol
func TestPing(t *testing.T) {
	slog.Warn(fmt.Sprintf("---Test: %s %s---\n", t.Name(), testProtocol))

	testEnv, cancelFn := testenv.StartTestEnv(testProtocol, true)
	defer cancelFn()
	// NewConsumerClient creates a client
	co1, cc1, _ := testEnv.NewConnectedConsumer(testClientID1, authn.ClientRoleViewer)
	defer cc1.Close()

	err := co1.Ping()
	require.NoError(t, err)
}

// Run a ping test with client cert auth for the given test protocol
func TestPingClientCert(t *testing.T) {
	slog.Warn(fmt.Sprintf("---Test: %s %s---\n", t.Name(), testProtocol))

	testEnv, cancelFn := testenv.StartTestEnv(testProtocol, true)
	defer cancelFn()

	// ensure the test client account exists
	err := testEnv.TestAuthn.AddClient(testClientID1, "test", authn.ClientRoleViewer)

	// NewConsumerClient creates a client
	// create a connection to the test server
	serverTD := testEnv.Server.GetTD()
	// cl, err := clients.NewTransportClient(
	// 	testEnv.ServerProtocol, testEnv.ServerURL, testEnv.CertBundle.CaCert)
	cl, err := clients.NewTransportClient(serverTD, td.HTOpPing, "", testEnv.CertBundle.RootCAs)
	require.NoError(t, err)
	cl.SetTimeout(time.Minute)
	err = cl.AuthenticateWithClientCert(testEnv.CertBundle.ClientCert)
	require.NoError(t, err)
	err = cl.Connect()
	require.NoError(t, err)
	status := cl.GetConnectionStatus()
	require.Equal(t, api.StatusConnected, status)

	cl.SetTimeout(time.Minute)
	defer cl.Close()

	// all hiveot transport handle a ping message
	req := msg.NewRequestMessage(td.HTOpPing, "", "", nil)
	err = cl.HandleRequest(req, func(resp *msg.ResponseMessage) error {
		slog.Info("Received response")
		return nil
	})
	require.NoError(t, err)

	status = cl.GetConnectionStatus()
	assert.Equal(t, api.StatusConnected, status)
}

// Test client returns UnauthorizedError when not authorized
func TestUnauthorizedError(t *testing.T) {
	slog.Warn(fmt.Sprintf("---Test: %s %s---\n", t.Name(), testProtocol))

	testEnv, cancelFn := testenv.StartTestEnv(testProtocol, true)
	defer cancelFn()

	tdoc := testEnv.Server.GetTD()
	connectForm, _ := tdoc.GetConnectForm("", "")
	rootCAs := testEnv.AppEnv.GetRootCAs()

	// ensure the test client account exists
	err := testEnv.TestAuthn.AddClient(testClientID1, "test", authn.ClientRoleViewer)
	token, _, err := testEnv.CreateToken(testClientID1, time.Minute*10)

	// first make sure connection does validate
	cl, err := clients.NewTransportClientFromForm(tdoc, connectForm, rootCAs)
	assert.NoError(t, err)
	err = cl.AuthenticateWithToken(testClientID1, token)
	assert.NoError(t, err)
	err = cl.Connect()
	assert.NoError(t, err)
	cl.Close()

	// check bad token - httpbasic doesnt detect this until a request is sent
	if testProtocol != api.HttpBasicProtocolType {
		err = cl.AuthenticateWithToken(testClientID1, "badtoken")
		assert.NoError(t, err)
		err = cl.Connect()
		assert.Equal(t, utils.UnauthorizedError, err)
		cl.Close()
	}

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
	slog.Warn(fmt.Sprintf("---Test: %s %s---\n", t.Name(), testProtocol))

	testEnv, cancelFn := testenv.StartTestEnv(testProtocol, true)
	defer cancelFn()
	serverURL := testEnv.Server.GetConnectURL()
	assert.NotEmpty(t, serverURL)
}

func TestServerTD(t *testing.T) {
	slog.Warn(fmt.Sprintf("---Test: %s %s---\n", t.Name(), testProtocol))

	testEnv, cancelFn := testenv.StartTestEnv(testProtocol, true)
	defer cancelFn()
	tdoc := testEnv.Server.GetTD()
	assert.NotEmpty(t, tdoc)
}
