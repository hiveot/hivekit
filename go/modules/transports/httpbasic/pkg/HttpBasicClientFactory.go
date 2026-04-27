package httpbasicpkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
)

// Create an HTTP-Basic client using the application environment from the provided factory
func NewHttpBasicClientFactory(f factory.IModuleFactory) modules.IHiveModule {

	env := f.GetEnvironment()
	m := NewHttpBasicClient(env.ServerURL, env.CaCert, nil, nil)
	clientCert, _ := env.GetClientCert()
	if clientCert != nil {
		m.ConnectWithClientCert(clientCert)
	}
	m.SetTimeout(env.RpcTimeout)
	return m
}
