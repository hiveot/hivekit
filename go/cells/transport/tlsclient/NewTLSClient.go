package tlsclient

import (
	"crypto/x509"
)

func NewTLSClient(serverURL string, rootCAs *x509.CertPool) ITLSClient {
	return NewTLSClientImpl(serverURL, rootCAs)
}
