package httptransportpkg

import (
	"crypto/x509"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	internal "github.com/hiveot/hivekit/go/modules/transports/httptransport/internal/client"
)

func NewHttpTransportClient(hostPort string, caCert *x509.Certificate, timeout time.Duration) transports.ITLSClient {
	return internal.NewTlsClient(hostPort, caCert, timeout)
}
