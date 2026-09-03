package tls_client

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/cells/transport/tlsclient"
	"github.com/hiveot/hivekit/go/cells/transport/tlsclient/internal"
)

func StartTLSClient(hostPort string, rootCAs *x509.CertPool) tlsclient.ITLSClient {
	return internal.StartTLSClientImpl(hostPort, rootCAs)
}
