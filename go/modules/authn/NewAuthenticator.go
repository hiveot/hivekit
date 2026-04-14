package authn

import (
	"log/slog"

	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules"
	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"
)

// Create an authenticator instance for use by transport modules
// func NewAuthenticator() transports.IAuthenticator {
// }

// Create a new instance of the authenticator using the factory.
// This uses the authn module authenticator.
// The resulting module can be converted to the IAuthenticator API.
func NewAuthenticatorFactory(f factoryapi.IModuleFactory) modules.IHiveModule {
	m, err := f.GetModule(authnapi.AuthnModuleType)
	if err != nil {
		slog.Error("NewAuthenticatorFactory: The default authenticator needs the authn module")
		return nil
	}
	return m
}
