package httpbasicapi

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/stretchr/testify/require"
)

func TestConnect(t *testing.T) {
	baseURL := "http://localhost:8080"
	clientID := "testclient"
	var caCert *x509.Certificate
	var sink modules.IHiveModule
	var timeout time.Duration
	var token = ""

	// TODO
	cl := NewHttpBasicClient(baseURL, caCert, sink, nil, timeout)

	err := cl.ConnectWithToken(clientID, token)
	require.NoError(t, err)

	cl.Close()
}
