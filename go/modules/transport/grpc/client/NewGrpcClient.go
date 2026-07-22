package grpc_client

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/transport/grpc/internal/clientimpl"
)

// NewHiveotGrpcClient creates a hiveot gRPC transport client.
//
// This uses the HiveOT RRN messages as the payload.
//
// addr is the UDS path or tcp connection to connect with
// caCert of the CA used for tcp URL's
//
// Use SetTimeout to change the default response timeout
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by exposed things.
func NewHiveotGrpcClient(
	addr string, caCert *x509.Certificate) api.ITransportClient {

	return clientimpl.NewGrpcClientImpl(addr, caCert)
}

// Create a hiveot gRPC client using the factory
func NewHiveotGrpcClientFactory(
	f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {

	var err error

	env := f.GetEnvironment()
	clientCert, _ := env.GetTLSCert()
	serverURL := env.GetServerURL()

	m := NewHiveotGrpcClient(serverURL, env.CaCert)
	m.SetTimeout(env.RpcTimeout)

	// if client certificate not available attempt auth token
	if clientCert == nil {
		// must use token auth
		clientID := env.GetClientID()
		authToken, err := env.GetAuthToken()

		if err == nil && clientID != "" && authToken != "" {
			err = m.AuthenticateWithToken(clientID, authToken)
		}
	}
	return m, err
}
