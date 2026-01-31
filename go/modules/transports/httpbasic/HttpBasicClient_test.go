package httpbasic_test

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	httpbasicclient "github.com/hiveot/hivekit/go/modules/transports/httpbasic/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnect(t *testing.T) {
	baseURL := "http://localhost:8080"
	clientID := "testclient"
	var caCert *x509.Certificate
	var timeout time.Duration
	var token = ""
	var isConnected = false

	// TODO
	cl := httpbasicclient.NewHttpBasicClient(baseURL, caCert, nil, timeout)

	err := cl.ConnectWithToken(clientID, token,
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
