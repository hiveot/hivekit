package tlsclient

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/api/msg"
)

func NewTLSClient(serverURL string, rootCAs *x509.CertPool) ITLSClient {
	return NewTLSClientImpl(serverURL, rootCAs, msg.CellTimeout)
}
