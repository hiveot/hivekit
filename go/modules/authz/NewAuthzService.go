package authz

import (
	"log/slog"

	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules"
	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"
	authzapi "github.com/hiveot/hivekit/go/modules/authz/api"
	"github.com/hiveot/hivekit/go/modules/authz/internal"
)

const AuthzModuleType = "authz"

func NewAuthzService(getRoleHandler func(clientID string) (role string, err error)) authzapi.IAuthzService {
	m := internal.NewAuthzService(getRoleHandler)
	return m
}

// factory function for creating authz module instance.
// This loads the authn module to use GetProfile to obtain the role.
func NewAuthzServiceFactory(f factoryapi.IModuleFactory) modules.IHiveModule {
	m1, err := f.GetModule(authnapi.AuthnModuleType)
	if err != nil {
		return nil
	}
	authn, ok := m1.(authnapi.IAuthnService)
	if !ok {
		slog.Error("Authz factory: cannot get authn service for obtaining roles")
		return nil
	}
	// getrole uses the authn module to get the client profile
	m := internal.NewAuthzService(func(clientID string) (string, error) {
		p, err := authn.GetProfile(clientID)
		if err != nil {
			return "", err
		}
		return p.Role, nil
	})
	return m
}
