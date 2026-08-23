package grpc_client

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/transport/grpc/internal/clientimpl"
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
	addr string, rootCAs *x509.CertPool) api.ITransportClient {

	return clientimpl.NewGrpcClientImpl(addr, rootCAs)
}

// Create a hiveot gRPC client using the factory
func NewHiveotGrpcClientFactory(
	f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	var err error

	env := f.GetEnvironment()
	clientCert, _ := env.GetClientCert()
	serverURL := env.ServerURL

	m := NewHiveotGrpcClient(serverURL, env.GetRootCAs())
	m.SetTimeout(env.RpcTimeout)

	// if client certificate not available attempt auth token
	if clientCert == nil {
		// must use token auth
		clientID := env.ClientID
		authToken, err := env.GetAuthToken()

		if err == nil && clientID != "" && authToken != "" {
			err = m.SetAuthToken(clientID, authToken, td.SecSchemeBearer)
		}
	}
	return m, err
}
