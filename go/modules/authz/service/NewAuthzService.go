package authz_service

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/authz"
	"github.com/hiveot/hivekit/go/modules/authz/internal"
)

const AuthzModuleType = "authz"

func NewAuthzService(getRoleHandler func(clientID string) (role string, err error)) authz.IAuthzService {
	m := internal.NewAuthzServiceImpl(getRoleHandler)
	return m
}

// factory function for creating authz module instance.
// This loads the authn module to use GetProfile to obtain the role.
func NewAuthzServiceFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	m1, err := f.StartModule(authn.AuthnServiceModuleType, true)
	if err != nil {
		return nil, err
	}
	authn, ok := m1.(authn.IAuthnService)
	if !ok {
		slog.Error("Authz factory: cannot get authn service for obtaining roles")
		return nil, err
	}
	// getrole uses the authn module to get the client profile
	m := internal.NewAuthzServiceImpl(func(clientID string) (string, error) {
		p, err := authn.GetProfile(clientID)
		if err != nil {
			return "", err
		}
		return p.Role, nil
	})
	return m, nil
}
