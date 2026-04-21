package grpctransportpkg

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/grpc/internal/grpcclient"
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
	addr string, clientCert *tls.Certificate, caCert *x509.Certificate, ch transports.ConnectionHandler) transports.ITransportClient {

	return grpcclient.NewGrpcTransportClient(addr, clientCert, caCert, ch)
}

// Create a gRPC client using the factory
func NewHiveotGrpcClientFactory(f factory.IModuleFactory) modules.IHiveModule {
	env := f.GetEnvironment()
	clientCert, _ := env.GetClientCert()
	serverURL := env.GetServerURL()

	m := grpcclient.NewGrpcTransportClient(serverURL, clientCert, env.CaCert, nil)
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
