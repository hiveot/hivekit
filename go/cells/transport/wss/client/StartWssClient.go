package wss_client

import (
	"crypto/x509"
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/transport/wss/internal/clientimpl"
)

// NewHiveotClient creates a new instance of the hiveot websocket client
// but does not connect yet. Authenticate and call Connect before use.
//
// This uses the Hiveot passthrough message converter.
//
//	wssURL is the full websocket connection URL including path
//	rootCAs are the CA's for TLS connection validation
//	ch is the connect/disconnect callback. nil to ignore
func StartHiveotWssClient(wssURL string, rootCAs *x509.CertPool) api.ITransportClient {

	return clientimpl.StartHiveotWssClientImpl(wssURL, rootCAs)
}

// Create a websocket client for the given factory environment
// Intended for devices that use reverse connections or consumer applications that
// use the factory. If the environment is setup with credentials then these are
// used to provision the client connection.
//
// This returns a transport client or an error if a valid authentication is missing
// Even with an error result authentication and 'Connect' can be called again.
func StartHiveotWssClientFactory(
	f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	var err error

	env := f.GetEnvironment()
	wssURL := env.ServerURL
	cl := StartHiveotWssClient(wssURL, env.GetRootCAs())
	cl.SetTimeout(env.RpcTimeout)

	// set client certificate if available
	clientCert, _ := env.GetClientCert()
	if clientCert != nil {
		err = cl.SetClientCert(clientCert)
	}
	// set auth token if available
	clientID := env.ClientID
	authToken, _ := env.GetAuthToken()

	if clientID != "" && authToken != "" {
		err = cl.SetAuthToken(clientID, authToken, td.SecSchemeBearer)
	}
	// connect
	if err == nil {
		err = cl.Connect()
	}
	if err != nil {
		slog.Error("StartHiveotWssClientFactory: " + err.Error())
	} else {
		slog.Info("StartHiveotWssClientFactory: connected", "clientID", clientID)
	}
	return cl, err
}

// StartWotWssClient creates a new instance of the WoT compatible websocket client
// but does not connect yet. Authenticate and call Connect before use.
//
// messageConverter offers the ability to use any websocket message format that
// can be mapped to a RequestMessage and ResponseMessage. It is used to support
// both hiveot and WoT websocket message formats.
//
// Users must use setAuthToken or SetClientCert to authenticate and Connect or Start
// to establish the connection.
//
//	wssURL is the full websocket connection URL
//	rootCAs are the server CA's for TLS connection validation
//	timeout is the maximum connection wait time. 0 for default.
//	ch is the connection callback handler, nil to ignore
func StartWotWssClient(
	wssURL string, rootCAs *x509.CertPool) api.ITransportClient {

	return clientimpl.StartWotWssClientImpl(wssURL, rootCAs)
}

// Create a websocket client for the given factory environment.
//
// This attempts to obtain server URL, authentication credentials if available and
// attempts to connect.
//
// Intended for devices that use reverse connections or consumer applications that
// use the factory. If the environment is setup with credentials then these are
// used to provision the client connection.
//
// This returns the client. If connection fails then this returns an error.
// Even with an error result authentication and 'Connect' can be called again.
func StartWotWssClientFactory(
	f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	var err error

	// create the client using the app env server URL
	env := f.GetEnvironment()
	serverURL := env.ServerURL
	cl := StartWotWssClient(serverURL, env.GetRootCAs())
	cl.SetTimeout(env.RpcTimeout)

	// set client certificate if available
	clientCert, _ := env.GetClientCert()
	if clientCert != nil {
		err = cl.SetClientCert(clientCert)
	}
	// set auth token if available
	clientID := env.ClientID
	authToken, _ := env.GetAuthToken()

	if clientID != "" && authToken != "" {
		err = cl.SetAuthToken(clientID, authToken, td.SecSchemeBearer)
	}
	// connect
	if err == nil {
		err = cl.Connect()
	}
	if err != nil {
		slog.Error("StartWotWssClientFactory: " + err.Error())
	}
	return cl, err
}
