package ssesc_client

import (
	"crypto/x509"
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/transport/ssesc/internal/clientimpl"
)

// NewSseScClient creates a new instance of the hiveot SSE-SC client.
//
//	sseURL is the full websocket connection URL including path
//	caCert is the server CA for TLS connection validation
//	ch is the connect/disconnect callback. nil to ignore
func NewSseScClient(sseURL string, caCert *x509.Certificate) api.ITransportClient {

	return clientimpl.NewSseScClientImpl(sseURL, caCert)
}

// Create an HTTP/SSE-SC client using the application environment from the provided factory
func NewSseScClientFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {

	env := f.GetEnvironment()
	// do clients use onconnectionchanged? -> yes, show connection status
	// how do they get informed? -> client submits an event
	clientCert, _ := env.GetTLSCert()
	m := NewSseScClient(env.ServerURL, env.CaCert)
	if clientCert != nil {
		err := m.AuthenticateWithClientCert(clientCert)
		if err != nil {
			slog.Error("NewSseScClientFactory. Failed: " + err.Error())
		}
	}
	m.SetTimeout(env.RpcTimeout)
	return m, nil
}
