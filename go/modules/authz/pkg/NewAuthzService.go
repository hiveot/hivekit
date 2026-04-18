package authzpkg

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/authz"
	"github.com/hiveot/hivekit/go/modules/authz/internal"
	"github.com/hiveot/hivekit/go/modules/factory"
)

const AuthzModuleType = "authz"

func NewAuthzService(getRoleHandler func(clientID string) (role string, err error)) authz.IAuthzService {
	m := internal.NewAuthzService(getRoleHandler)
	return m
}

// factory function for creating authz module instance.
// This loads the authn module to use GetProfile to obtain the role.
func NewAuthzServiceFactory(f factory.IModuleFactory) modules.IHiveModule {
	m1, err := f.GetModule(authn.AuthnModuleType)
	if err != nil {
		return nil
	}
	authn, ok := m1.(authn.IAuthnService)
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
