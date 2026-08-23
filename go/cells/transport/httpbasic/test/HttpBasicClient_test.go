package httpbasic_test

import (
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	certstest "github.com/hiveot/hivekit/go/cells/certs/test"
	httpbasic_client "github.com/hiveot/hivekit/go/cells/transport/httpbasic/client"
	httpbasic_server "github.com/hiveot/hivekit/go/cells/transport/httpbasic/server"
	"github.com/hiveot/hivekit/go/cells/transport/tlsserver"
	tls_server "github.com/hiveot/hivekit/go/cells/transport/tlsserver/server"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var serverPort int = 9445
var testCerts = certstest.CreateTestCertBundle(utils.KeyTypeED25519)
var rpcTimeout = time.Minute * 3 // for debugging

func TestConnect(t *testing.T) {
	clientID := "testclient"
	var token = ""

	// first start the server
	testAuthenticator := testenv.NewTestAuthenticator()
	cfg := tlsserver.NewTLSServerConfig(
		"localhost", serverPort, testCerts.ServerCert, testCerts.RootCAs, true)
	srv := tls_server.NewTLSServer(cfg, testAuthenticator)
	err := srv.Start()

	require.NoError(t, err)
	m := httpbasic_server.NewHttpBasicServer(srv)
	err = m.Start()
	// this could work if all servers have a TD
	tdoc := m.GetTD()
	require.NoError(t, err)

	// get the client
	cl := httpbasic_client.NewHttpBasicClient(tdoc, testCerts.RootCAs)
	cl.SetTimeout(rpcTimeout)
	err = cl.SetAuthToken(clientID, token, td.SecSchemeBearer)
	require.NoError(t, err)
	err = cl.Connect()
	require.NoError(t, err)
	assert.Equal(t, api.StatusConnected, cl.GetConnectionStatus())

	cl.Close()
	time.Sleep(time.Millisecond)
	assert.Equal(t, api.StatusClosed, cl.GetConnectionStatus())
}
