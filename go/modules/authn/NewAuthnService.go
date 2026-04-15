package authn

import (
	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules"
	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"

	"github.com/hiveot/hivekit/go/modules/authn/internal/service"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// NewAuthnService create a new instance of the authentication service.
// This service offers the ability to manage clients.
//
// Note: to avoid a chicken-and-egg problem between authentication and http server,
// create the http server first and pass it to the authenticator. The authenticator will
// invoke httpserver.SetAuthValidator on start.
//
// authnConfig contains the password storage and token management configuration
// httpServer to server the http endpoint or nil to not use http.
func NewAuthnService(
	authnConfig authnapi.AuthnConfig,
	httpServer transports.IHttpServer) authnapi.IAuthnService {

	m := service.NewAuthnService(authnConfig, httpServer)
	return m
}

// Create a new instance of the authentication service using the factory environment.
// The factory will provide the configuration and http server.
// This sets the authn session manager as the factory authenticator.
func NewAuthnServiceFactory(f factoryapi.IModuleFactory) modules.IHiveModule {
	env := f.GetEnvironment()
	keysDir := env.CertsDir
	storageDir := env.GetStorageDir(authnapi.AuthnModuleType)
	authnConfig := authnapi.NewAuthnConfig(keysDir, storageDir)
	// TODO: configuration for using http endpoints in authn
	httpServer := f.GetHttpServer()
	m := NewAuthnService(authnConfig, httpServer)
	f.SetAuthenticator(m.GetSessionManager())
	return m
}
