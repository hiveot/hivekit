package wsspkg

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
)

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
