package authn_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/authn/internal/serviceimpl"
)

// admin auth validity
const DefaultAdminTokenValidityDays = 366

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

	svc := serviceimpl.NewAuthnServiceImpl(authnConfig)
	return svc
}

// Create a new instance of the authentication service using the factory environment.
//
// The factory will provide the configuration and http server.
// This sets the authn session manager as the factory authenticator.
// This configures the authn service to create an admin account token on startup.
func NewAuthnServiceFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	env := f.GetEnvironment()
	keysDir := env.CertsDir
	storageDir := env.GetStorageDir(authn.AuthnServiceModuleType)
	authnConfig := authn.NewAuthnConfig(keysDir, storageDir)
	authnConfig.AdminTokenValidityDays = DefaultAdminTokenValidityDays
	svc := NewAuthnService(authnConfig)
	f.SetAuthenticator(svc.GetSessionManager())
	return svc, nil
}
