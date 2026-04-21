package httpclient

import (
	"crypto/x509"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpclient/internal"
)

func NewHttpClient(hostPort string, caCert *x509.Certificate, timeout time.Duration) transports.ITLSClient {
	return internal.NewHttpClient(hostPort, caCert, timeout)
}
