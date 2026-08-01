package tls_client

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/modules/transport/tlsclient"
	"github.com/hiveot/hivekit/go/modules/transport/tlsclient/internal"
)

func NewTLSClient(hostPort string, rootCAs *x509.CertPool) tlsclient.ITLSClient {
	return internal.NewTLSClientImpl(hostPort, rootCAs)
}
