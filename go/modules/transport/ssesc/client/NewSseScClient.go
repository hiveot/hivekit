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
//	rootCAs are CA certificates to validate the server certificate. nil for system CAs.
//	ch is the connect/disconnect callback. nil to ignore
func NewSseScClient(sseURL string, rootCAs *x509.CertPool) api.ITransportClient {

	return clientimpl.NewSseScClientImpl(sseURL, rootCAs)
}

// Create an HTTP/SSE-SC client using the application environment from the provided factory
func NewSseScClientFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {

	env := f.GetEnvironment()
	// do clients use onconnectionchanged? -> yes, show connection status
	// how do they get informed? -> client submits an event
	clientCert, _ := env.GetClientCert()
	m := NewSseScClient(env.ServerURL, env.GetRootCAs())
	if clientCert != nil {
		err := m.SetClientCert(clientCert)
		if err != nil {
			slog.Error("NewSseScClientFactory. Failed: " + err.Error())
		}
	}
	m.SetTimeout(env.RpcTimeout)
	return m, nil
}
