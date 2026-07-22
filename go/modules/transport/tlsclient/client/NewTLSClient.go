package tls_client

import (
	"crypto/x509"
	"time"

	"github.com/hiveot/hivekit/go/modules/transport/tlsclient"
	"github.com/hiveot/hivekit/go/modules/transport/tlsclient/internal"
)

func NewTLSClient(hostPort string, caCert *x509.Certificate, timeout time.Duration) tlsclient.ITLSClient {
	return internal.NewTLSClientImpl(hostPort, caCert, timeout)
}
