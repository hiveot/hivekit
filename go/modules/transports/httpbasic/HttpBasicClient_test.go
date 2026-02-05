package httpbasic_test

import (
	"crypto/x509"
	"fmt"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/certs/module/selfsigned"
	"github.com/hiveot/hivekit/go/modules/transports"
	httpbasicclient "github.com/hiveot/hivekit/go/modules/transports/httpbasic/client"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic/httpbasicserver"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver/module"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var serverPort int = 9445
var testCerts = selfsigned.CreateTestCertBundle(utils.KeyTypeED25519)

func TestConnect(t *testing.T) {
	baseURL := fmt.Sprintf("http://localhost:%d", serverPort)
	clientID := "testclient"
	var caCert *x509.Certificate
	var token = ""
	var isConnected = false

	// dummyAuthenticator := authnapi.NewDummyAuthenticator()
	cfg := httpserver.NewHttpServerConfig(
		"localhost", serverPort, testCerts.ServerCert, testCerts.CaCert, nil)
	srv := module.NewHttpServerModule("", cfg)
	err := srv.Start()
	require.NoError(t, err)
	m := httpbasicserver.NewHttpBasicServer(srv)

	err = m.Start("")
	require.NoError(t, err)

	cl := httpbasicclient.NewHttpBasicClient(baseURL, caCert, nil)
	err = cl.ConnectWithToken(clientID, token,
		func(connected bool, cl2 transports.IConnection, err2 error) {
			isConnected = connected
		})
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	assert.True(t, isConnected)

	cl.Close()
	time.Sleep(time.Millisecond)
	assert.False(t, isConnected)
}
