package authz_service

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/authn"
	"github.com/hiveot/hivekit/go/cells/authz"
	"github.com/hiveot/hivekit/go/cells/authz/internal"
)

const AuthzCellType = "authz"

// Create a new ready-to-use authz service instance.
// Call Start to publish a TD.
func NewAuthzService(getRoleHandler func(clientID string) (role string, err error)) authz.IAuthzService {
	svc := internal.NewAuthzServiceImpl(getRoleHandler)
	return svc
}

// factory function for creating authz service instance.
// This loads the authn service to use GetProfile to obtain the role.
func NewAuthzServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	m1, err := f.NewCell(authn.AuthnServiceCellType, true)
	if err != nil {
		return nil, err
	}
	authn, ok := m1.(authn.IAuthnService)
	if !ok {
		slog.Error("Authz factory: cannot get authn service for obtaining roles")
		return nil, err
	}
	// getrole uses the authn service to get the client profile
	svc := internal.NewAuthzServiceImpl(func(clientID string) (string, error) {
		p, err := authn.GetProfile(clientID)
		if err != nil {
			return "", err
		}
		return p.Role, nil
	})
	return svc, nil
}
