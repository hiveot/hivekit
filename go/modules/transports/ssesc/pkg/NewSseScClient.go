package ssescpkg

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/modules/transports"
	internal "github.com/hiveot/hivekit/go/modules/transports/ssesc/internal/client"
)

// NewSseScClient creates a new instance of the hiveot SSE-SC client.
//
//	sseURL is the full websocket connection URL including path
//	caCert is the server CA for TLS connection validation
//	ch is the connect/disconnect callback. nil to ignore
func NewSseScClient(sseURL string, caCert *x509.Certificate,
	ch transports.ConnectionHandler) transports.ITransportClient {

	return internal.NewSseScClient(sseURL, caCert, ch)
}
