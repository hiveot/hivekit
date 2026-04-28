package grpcpkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
)

// Create a hiveot gRPC client using the factory
func NewHiveotGrpcClientFactory(f factory.IModuleFactory) modules.IHiveModule {
	env := f.GetEnvironment()
	clientCert, _ := env.GetClientCert()
	serverURL := env.GetServerURL()

	m := NewHiveotGrpcClient(serverURL, env.CaCert, nil)
	m.SetTimeout(env.RpcTimeout)

	// if client certificate not available attempt auth token
	if clientCert == nil {
		// must use token auth
		clientID := env.GetClientID()
		authToken := env.GetAuthToken()

		if clientID != "" && authToken != "" {
			m.ConnectWithToken(clientID, authToken)
		}
	}
	return m
}
