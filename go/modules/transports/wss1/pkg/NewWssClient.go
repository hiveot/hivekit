package wsspkg

import (
	"crypto/x509"
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/wss1/internal/client"
)

// NewHiveotWssClient creates a new instance of the hiveot websocket client.
//
// This uses the Hiveot passthrough message converter.
//
//	wssURL is the full websocket connection URL including path
//	caCert is the server CA for TLS connection validation
//	ch is the connect/disconnect callback. nil to ignore
func NewHiveotWssClient(wssURL string, caCert *x509.Certificate,
	ch transports.ConnectionHandler) transports.ITransportClient {

	return client.NewHiveotWssClient(wssURL, caCert, ch)
}

// Create a websocket client for the given factory environment
// Intended for devices that use reverse connections or consumer applications that
// use the factory. If the environment is setup with credentials then these are
// used to provision the client connection.
func NewHiveotWssClientFactory(f factory.IModuleFactory) modules.IHiveModule {
	var err error

	env := f.GetEnvironment()
	clientCert, _ := env.GetClientCert()
	wssURL := env.GetServerURL()
	m := NewHiveotWssClient(wssURL, env.CaCert, nil)
	m.SetTimeout(env.RpcTimeout)
	if clientCert != nil {
		err = m.ConnectWithClientCert(clientCert)
	} else {
		// if client certificate not available attempt auth token
		clientID := env.GetClientID()
		authToken := env.GetAuthToken()

		if clientID != "" && authToken != "" {
			m.ConnectWithToken(clientID, authToken)
		}
	}
	if err != nil {
		slog.Error("NewWotWssClientFactory: " + err.Error())
	}
	return m
}

// NewWotWssClient creates a new instance of the WoT compatible websocket client.
//
// messageConverter offers the ability to use any websocket message format that
// can be mapped to a RequestMessage and ResponseMessage. It is used to support
// both hiveot and WoT websocket message formats.
//
// Users must use ConnectWithToken to authenticate and connect.
//
//	wssURL is the full websocket connection URL
//	caCert is the server CA for TLS connection validation
//	timeout is the maximum connection wait time. 0 for default.
//	ch is the connection callback handler, nil to ignore
func NewWotWssClient(wssURL string, caCert *x509.Certificate,
	ch transports.ConnectionHandler) transports.ITransportClient {
	return client.NewWotWssClient(wssURL, caCert, ch)
}

// Create a websocket client for the given factory environment.
// This attempts to obtain server URL and authentication credentials if available.
//
// Intended for devices that use reverse connections or consumer applications that
// use the factory. If the environment is setup with credentials then these are
// used to provision the client connection.
func NewWotWssClientFactory(f factory.IModuleFactory) modules.IHiveModule {

	var err error

	env := f.GetEnvironment()
	clientCert, _ := env.GetClientCert()
	serverURL := env.GetServerURL()

	m := NewWotWssClient(serverURL, env.CaCert, nil)
	m.SetTimeout(env.RpcTimeout)
	// if client certificate not available attempt auth token
	if clientCert != nil {
		err = m.ConnectWithClientCert(clientCert)
	} else {
		// must use token auth
		clientID := env.GetClientID()
		authToken := env.GetAuthToken()

		if clientID != "" && authToken != "" {
			err = m.ConnectWithToken(clientID, authToken)
		}
	}
	if err != nil {
		slog.Error("NewWotWssClientFactory: " + err.Error())
	}
	return m
}
