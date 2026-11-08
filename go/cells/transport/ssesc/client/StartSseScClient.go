package ssesc_client

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/transport/ssesc/client/internal"
)

// NewSseScClient creates a ready-to-use instance of the hiveot SSE-SC client.
//
// Set the authentication credentials and call Connect() or Start().
//
//	tdoc containing the http URLs for operations. Base contains the sse connect URL.
//	rootCAs are CA certificates to validate the server certificate. nil for system CAs.
func NewSseScClient(tdoc *wot.TD, rootCAs *x509.CertPool) api.ITransportClient {

	return internal.NewSseScClientImpl(tdoc, rootCAs)
}

// // Create a ready-to-use HTTP/SSE-SC client using the application environment to set
// // the server URL and authentication client-certificate from the provided factory.
// func NewSseScClientFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

// 	env := f.GetEnvironment()
// 	// do clients use onconnectionchanged? -> yes, show connection status
// 	// how do they get informed? -> client submits an event
// 	clientCert, _ := env.GetClientCert()
// 	m := NewSseScClient(env.GetServerTD(), env.GetRootCAs())
// 	if clientCert != nil {
// 		err := m.SetClientCert(clientCert)
// 		if err != nil {
// 			slog.Error("NewSseScClientFactory. Failed: " + err.Error())
// 		}
// 	}
// 	return m, nil
// }
