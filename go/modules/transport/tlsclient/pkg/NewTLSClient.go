package tlsclientpkg

import (
	"crypto/x509"
	"time"

	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/tlsclient/internal"
)

func NewTLSClient(hostPort string, caCert *x509.Certificate, timeout time.Duration) transport.ITLSClient {
	return internal.NewTLSClient(hostPort, caCert, timeout)
}
