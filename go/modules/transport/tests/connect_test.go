package transporttests

import (
	"log/slog"
	"os"
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
		t.Run(testProtocol, TestServerURL)
	}
}

// test create a server and connect a client
func TestStartStop(t *testing.T) {
	t.Logf("---%s %s---\n", t.Name(), testProtocol)

	// testenv might still start the httpserver - fixme: use on-demand factory
	testEnv, cancelFn := testenv.StartTestEnv(testProtocol, true)

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

	testEnv, cancelFn := testenv.StartTestEnv(testProtocol, true)
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

	testEnv, cancelFn := testenv.StartTestEnv(testProtocol, true)
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
	status := cl.GetConnectionStatus()
	require.Equal(t, transport.StatusConnected, status)

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
	assert.Equal(t, transport.StatusConnected, status)
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

	testEnv, cancelFn := testenv.StartTestEnv(testProtocol, true)
	defer cancelFn()
	serverURL := testEnv.Server.GetConnectURL()
	assert.NotEmpty(t, serverURL)
}
