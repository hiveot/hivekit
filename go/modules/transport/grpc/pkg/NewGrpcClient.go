package grpcpkg

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transport"
	internalclient "github.com/hiveot/hivekit/go/modules/transport/grpc/internal/client"
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
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewHiveotGrpcClient(
	addr string, caCert *x509.Certificate) transport.ITransportClient {

	return internalclient.NewGrpcClient(addr, caCert)
}

// Create a hiveot gRPC client using the factory
func NewHiveotGrpcClientFactory(f factory.IModuleFactory) modules.IHiveModule {
	env := f.GetEnvironment()
	clientCert, _ := env.GetClientCert()
	serverURL := env.GetServerURL()

	m := NewHiveotGrpcClient(serverURL, env.CaCert)
	m.SetTimeout(env.RpcTimeout)

	// if client certificate not available attempt auth token
	if clientCert == nil {
		// must use token auth
		clientID := env.GetClientID()
		authToken := env.GetAuthToken()

		if clientID != "" && authToken != "" {
			m.AuthenticateWithToken(clientID, authToken)
		}
	}
	return m
}
