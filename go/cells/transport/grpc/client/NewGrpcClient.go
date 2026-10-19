package grpc_client

import (
	"crypto/x509"
	"strings"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/transport/grpc/client/internal"
)

// NewHiveotGrpcClient returns a ready-to-use hiveot gRPC transport client.
//
// Authenticate and call Connect before use.
//
// addr is the UDS path or tcp connection to connect with
// caCert of the CA used for tcp URL's
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by exposed things.
func NewHiveotGrpcClient(
	addr string, rootCAs *x509.CertPool) api.ITransportClient {

	return internal.NewGrpcClientImpl(addr, rootCAs)
}

// NewHiveotGrpcClientFactory returns a ready-to-use hiveot gRPC client using
// the cell factory for the client ID, auth certificate or auth token, and
// ServerURL for the endpoint to connect to.
//
// This requires the Server URL and authentication credentials. If serverURL is not set
// then try to obtain the directory URL instead. If both fail then this returns nil and
// the cell will be ignored in recipes.
func NewHiveotGrpcClientFactory(
	f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	var err error

	env := f.GetEnvironment()
	clientCert, _ := env.GetClientCert()
	serverURL := env.ServerURL
	if serverURL == "" {
		dirTD := env.GetDirTD()
		if dirTD != nil {
			form, match := dirTD.GetForm("", "", api.HiveotGrpcTcpProtocolType, api.HiveotGrpcTcpSubprotocol)
			if match {
				serverURL, _, err = dirTD.ResolveHRef(form, nil)
			}
		}
	}

	if serverURL == "" {
		// return nil without error to ignore this cell in a recipe
		return nil, nil
	}

	if !strings.HasPrefix(serverURL, api.HiveotGrpcUnixScheme) &&
		!strings.HasPrefix(serverURL, api.HiveotGrpcTcpScheme) {
		// there is a serverURL but it isnt for gRPC
		return nil, nil
	}

	cl := NewHiveotGrpcClient(serverURL, env.GetRootCAs())

	// if client certificate not available attempt auth token
	if clientCert == nil {
		// must use token auth
		clientID := env.ClientID
		authToken, err := env.GetAuthToken()

		if err == nil && clientID != "" && authToken != "" {
			err = cl.SetAuthToken(clientID, authToken, td.SecSchemeBearer)
		}
	}
	return cl, err
}
