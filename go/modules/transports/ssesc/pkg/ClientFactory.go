package ssescpkg

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	factory "github.com/hiveot/hivekit/go/modules/factory"
)

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
