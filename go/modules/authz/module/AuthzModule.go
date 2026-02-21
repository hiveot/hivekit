package module

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/authz/server"
	"github.com/hiveot/hivekit/go/msg"
)

// AuthzModule is a module for role based authorization of requests.
//
// This implements IHiveModule and IAuthzModule interfaces and is facade for the
// authorization store. This uses the authenticator provided client role as the role
// for RBAC.
type AuthzModule struct {
	modules.HiveModuleBase

	// config authz.AuthzConfig

	// store with authorization rules for devices
	authn authn.IAuthnModule
}

// Handle requests to be served by this module and filter unauthorized requests.
// This depends on a validated SenderID in the request message.
func (m *AuthzModule) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	err := m.ValidateAuthorization(req)
	if err != nil {
		return err
	}

	if req.ThingID == server.AuthzAdminServiceID {
		return server.HandleAuthzAdminRequest(m, req, replyTo)
	}
	// forward the request to the chain
	return m.HiveModuleBase.HandleRequest(req, replyTo)
}

// start opens the store with authorization rules
// currently the RBAC is hard coded so nothing to configure
func (m *AuthzModule) Start(yamlConfig string) (err error) {
	if m.authn == nil {
		slog.Warn("AuthzModule: no authenticator provided, only read requests will be accepted")
	}
	return nil
}

// Stop closes the rules store and releases resources
func (m *AuthzModule) Stop() {
}

// Create a new instance of the authorization module.
// The authenticator is used to get the client role for authorization decisions.
// Without the authenticator only read requests are accepted.
func NewAuthzModule(authenticator authn.IAuthnModule) *AuthzModule {
	m := &AuthzModule{
		authn: authenticator,
	}
	return m
}
