package httpbasic_test

import (
	"crypto/x509"
	"fmt"
	"testing"
	"time"

	certstest "github.com/hiveot/hivekit/go/modules/certs/test"
	"github.com/hiveot/hivekit/go/modules/transport"
	httpbasicpkg "github.com/hiveot/hivekit/go/modules/transport/httpbasic/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/httptransport"
	httptransportpkg "github.com/hiveot/hivekit/go/modules/transport/httptransport/pkg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var serverPort int = 9445
var testCerts = certstest.CreateTestCertBundle(utils.KeyTypeED25519)
var rpcTimeout = time.Minute * 3 // for debugging

func TestConnect(t *testing.T) {
	baseURL := fmt.Sprintf("http://localhost:%d", serverPort)
	clientID := "testclient"
	var caCert *x509.Certificate
	var token = ""

	// dummyAuthenticator := authnapi.NewDummyAuthenticator()
	cfg := httptransport.NewConfig(
		"localhost", serverPort, testCerts.ServerCert, testCerts.CaCert, true)
	srv := httptransportpkg.NewHttpTransportServer(cfg, nil)
	err := srv.Start()
	require.NoError(t, err)
	m := httpbasicpkg.NewHttpBasicServer(srv)

	err = m.Start()
	require.NoError(t, err)

	cl := httpbasicpkg.NewHttpBasicClient(baseURL, caCert, nil)
	cl.SetTimeout(rpcTimeout)
	err = cl.AuthenticateWithToken(clientID, token)
	require.NoError(t, err)
	err = cl.Connect()
	require.NoError(t, err)
	assert.Equal(t, transport.StatusConnected, cl.GetConnectionStatus())

	cl.Close()
	time.Sleep(time.Millisecond)
	assert.Equal(t, transport.StatusClosed, cl.GetConnectionStatus())
}
