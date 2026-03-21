package httpbasic_test

import (
	"crypto/x509"
	"fmt"
	"testing"
	"time"

	certstest "github.com/hiveot/hivekit/go/modules/certs/test"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic"
	httpbasicclient "github.com/hiveot/hivekit/go/modules/transports/httpbasic/client"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	httpserverapi "github.com/hiveot/hivekit/go/modules/transports/httpserver/api"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var serverPort int = 9445
var testCerts = certstest.CreateTestCertBundle(utils.KeyTypeED25519)

func TestConnect(t *testing.T) {
	baseURL := fmt.Sprintf("http://localhost:%d", serverPort)
	clientID := "testclient"
	var caCert *x509.Certificate
	var token = ""
	var isConnected = false

	// dummyAuthenticator := authnapi.NewDummyAuthenticator()
	cfg := httpserverapi.NewConfig(
		"localhost", serverPort, testCerts.ServerCert, testCerts.CaCert, nil)
	srv := httpserver.NewHttpServerModule(cfg)
	err := srv.Start()
	require.NoError(t, err)
	m := httpbasic.NewTransport(srv)

	err = m.Start("")
	require.NoError(t, err)

	cl := httpbasicclient.NewHttpBasicClient(baseURL, caCert, nil,
		func(connected bool, cl2 transports.IConnection, err2 error) {
			isConnected = connected
		})
	err = cl.ConnectWithToken(clientID, token)
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	assert.True(t, isConnected)

	cl.Close()
	time.Sleep(time.Millisecond)
	assert.False(t, isConnected)
}
