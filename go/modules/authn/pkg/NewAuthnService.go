package authnpkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/factory"

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
	authnConfig authn.AuthnConfig,
	httpServer transports.IHttpServer) authn.IAuthnService {

	m := service.NewAuthnService(authnConfig, httpServer)
	return m
}

// Create a new instance of the authentication service using the factory environment.
// The factory will provide the configuration and http server.
// This sets the authn session manager as the factory authenticator.
func NewAuthnServiceFactory(f factory.IModuleFactory) modules.IHiveModule {
	env := f.GetEnvironment()
	keysDir := env.CertsDir
	storageDir := env.GetStorageDir(authn.AuthnModuleType)
	authnConfig := authn.NewAuthnConfig(keysDir, storageDir)
	// TODO: option to enable/disable the authn http endpoints
	httpServer := f.GetHttpServer()
	m := NewAuthnService(authnConfig, httpServer)
	f.SetAuthenticator(m.GetSessionManager())
	return m
}
