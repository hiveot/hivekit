package httptransportpkg

import (
	"crypto/x509"
	"time"

	"github.com/hiveot/hivekit/go/modules/transport"
	internal "github.com/hiveot/hivekit/go/modules/transport/httptransport/internal/client"
)

func NewHttpTransportClient(hostPort string, caCert *x509.Certificate, timeout time.Duration) transport.ITLSClient {
	return internal.NewTlsClient(hostPort, caCert, timeout)
}
