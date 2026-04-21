package ssescpkg

import (
	"crypto/x509"
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	factory "github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/ssesc/internal/client"
)

// NewSseScClient creates a new instance of the hiveot SSE-SC client.
//
//	sseURL is the full websocket connection URL including path
//	caCert is the server CA for TLS connection validation
//	ch is the connect/disconnect callback. nil to ignore
func NewSseScClient(sseURL string, caCert *x509.Certificate,
	ch transports.ConnectionHandler) transports.ITransportClient {

	return client.NewHiveotSseClient(sseURL, caCert, ch)
}

// Create an HTTP/SSE-SC client using the application environment from the provided factory
func NewSseScClientFactory(f factory.IModuleFactory) modules.IHiveModule {

	env := f.GetEnvironment()
	// do clients use onconnectionchanged? -> yes, show connection status
	// how do they get informed? -> client submits an event
	clientCert, _ := env.GetClientCert()
	m := NewSseScClient(env.ServerURL, env.CaCert, nil)
	if clientCert != nil {
		err := m.ConnectWithClientCert(clientCert)
		if err != nil {
			slog.Error("NewSseScClientFactory. Failed: " + err.Error())
		}
	}
	m.SetTimeout(env.RpcTimeout)
	return m
}
