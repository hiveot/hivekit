package authnpkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/factory"

	"github.com/hiveot/hivekit/go/modules/authn/internal/service"
)

// NewAuthnService create a new instance of the authentication service using RRN messaging.
// This service offers the ability to manage clients.
//
// To support the http auth endpoint first start pkg.NewAuthnHttpService and link
// it to this module.
//
// authnConfig contains the password storage and token management configuration
// httpServer to server the http endpoint or nil to not use http.
func NewAuthnService(
	authnConfig authn.AuthnConfig) authn.IAuthnService {

	m := service.NewAuthnService(authnConfig)
	return m
}

// Create a new instance of the authentication service using the factory environment.
// The factory will provide the configuration and http server.
// This sets the authn session manager as the factory authenticator.
func NewAuthnServiceFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	env := f.GetEnvironment()
	keysDir := env.CertsDir
	storageDir := env.GetStorageDir(authn.AuthnServiceModuleType)
	authnConfig := authn.NewAuthnConfig(keysDir, storageDir)
	m := NewAuthnService(authnConfig)
	f.SetAuthenticator(m.GetSessionManager())
	return m, nil
}
