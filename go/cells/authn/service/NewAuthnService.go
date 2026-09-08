package authn_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/authn"
	"github.com/hiveot/hivekit/go/cells/authn/internal/serviceimpl"
)

// admin auth validity
const DefaultAdminTokenValidityDays = 366

// NewAuthnService returns a ready-to-use authentication service instance.
// This service offers the ability to manage clients.
//
// To support the http auth endpoint first start pkg.NewAuthnHttpService and link
// it to this service.
//
// authnConfig contains the password storage and token management configuration
func NewAuthnService(
	authnConfig authn.AuthnConfig) (authn.IAuthnService, error) {

	svc, err := serviceimpl.NewAuthnServiceImpl(authnConfig)
	return svc, err
}

// Return a ready-to-use instance of the authentication service using the factory environment.
//
// The factory environment is used to provide the configuration.
// This sets the authn session manager as the factory authenticator.
// This configures the authn service to create an admin account token on startup.
func NewAuthnServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	env := f.GetEnvironment()
	keysDir := env.CertsDir
	storageDir := env.GetStorageDir(authn.AuthnServiceCellType)

	authnConfig := authn.NewAuthnConfig(keysDir, storageDir)
	authnConfig.AdminTokenValidityDays = DefaultAdminTokenValidityDays

	svc, err := NewAuthnService(authnConfig)
	if err != nil {
		return nil, err
	}
	f.SetAuthenticator(svc.GetSessionManager())
	return svc, err
}
