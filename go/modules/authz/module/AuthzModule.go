package module

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/authn/server"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
)

// AuthzModule is a module for role based authorization of requests.
//
// This implements IHiveModule and IAuthzModule interfaces and is facade for the
// authorization store. This uses the authenticator provided client role as the role
// for RBAC.
type AuthzModule struct {
	modules.HiveModuleBase

	config authz.AuthzConfig

	// The authenticator providing roles
	authenticator transports.IAuthenticator
	// store with authorization rules for devices
	authzStore authzstore.IAuthzStore
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
	} else {
		// forward
		return m.HiveModuleBase.HandleRequest(req, replyTo)
	}
}

// start opens the store with authorization rules
func (m *AuthzModule) Start(yamlConfig string) (err error) {
}

// Stop closes the rules store and releases resources
func (m *AuthzModule) Stop() {
}

// ValidateAuthorization verifies that the sender is authorized for the request
func (m *AuthzModule) ValidateAuthorization(req *msg.RequestMessage) (err error) {
	clientID, role := m.authenticator.GetProfile()
	return err
}

func NewAuthzModule(authenticator transports.IAuthenticator) *AuthzModule {
	m := &AuthzModule{
		authenticator: authenticator,
	}
	return m
}
