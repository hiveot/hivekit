package authn_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/authn"
	"github.com/hiveot/hivekit/go/cells/authn/service/internal"
)

// NewAuthnService returns a ready-to-use authentication service instance.
// This service includes two nested Things, one for administrators  to manage clients and one for end users.
//
//	storageDir where to store authn data
//	tokenDir where to store authentication tokens; eg certs dir
//	createAdminAccount flag to create a default admin account
func NewAuthnService(
	tokensDir string, storageDir string, createAdminAcct bool) (authn.IAuthnService, error) {

	svc, err := internal.NewAuthnServiceImpl(tokensDir, storageDir, createAdminAcct)
	return svc, err
}

// Return a ready-to-use instance of the authentication service using the factory environment.
// This also creates an admin account with token file.
//
// The factory environment is used to provide the configuration.
// This configures the authn service to create an admin account token on startup.
func NewAuthnServiceFactory(
	f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	var err error

	env := f.GetEnvironment()
	tokensDir := env.CertsDir
	storageDir := env.GetStorageDir(authn.AuthnServiceCellType)
	svc, err := NewAuthnService(tokensDir, storageDir, true)
	if err != nil {
		return nil, err
	}
	userSvc := svc.GetUserService()
	f.SetAuthenticator(userSvc.GetSessionManager())
	return svc, err
}
