package httpbasic_test

import (
	"crypto/x509"
	"testing"
	"time"

	httpbasicclient "github.com/hiveot/hivekit/go/modules/transports/httpbasic/client"
	"github.com/stretchr/testify/require"
)

func TestConnect(t *testing.T) {
	baseURL := "http://localhost:8080"
	clientID := "testclient"
	var caCert *x509.Certificate
	var timeout time.Duration
	var token = ""

	// TODO
	cl := httpbasicclient.NewHttpBasicClient(baseURL, caCert, nil, timeout)

	err := cl.ConnectWithToken(clientID, token)
	require.NoError(t, err)

	cl.Close()
}
